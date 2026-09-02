package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Record describes one panel backup. The archive intentionally excludes the
// install directory; SteamCMD can recreate the game files at any time.
type Record struct {
	ID                    string    `json:"id"`
	Note                  string    `json:"note"`
	CreatedAt             time.Time `json:"created_at"`
	GameVersion           string    `json:"game_version"`
	Archive               string    `json:"archive"`
	SizeBytes             int64     `json:"size_bytes"`
	PanelSettingsIncluded bool      `json:"panel_settings_included,omitempty"`
}

type Service struct {
	BaseDir  string
	PanelDir string
	mu       sync.Mutex
}

var includedPaths = []string{"Server", "Saves", "db", "Workshop"}
var includedFiles = []string{"options.ini", "latestSave.ini"}

func NewService(baseDir string) *Service { return &Service{BaseDir: baseDir} }

func (s *Service) backupDir() string { return filepath.Join(s.BaseDir, "backups", "panel") }

func (s *Service) Create(note, gameVersion string, maxVersions int) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.backupDir(), 0o700); err != nil {
		return Record{}, err
	}
	now := time.Now().UTC()
	id := now.Format("20060102T150405.000000000Z")
	note = sanitizeNote(note)
	panelSettingsPath := filepath.Join(s.PanelDir, "panel_settings.json")
	_, panelErr := os.Stat(panelSettingsPath)
	record := Record{ID: id, Note: note, CreatedAt: now, GameVersion: gameVersion, Archive: "backup-" + id + ".tar.gz", PanelSettingsIncluded: s.PanelDir != "" && panelErr == nil}
	tmp := filepath.Join(s.backupDir(), "."+record.Archive+".tmp")
	archivePath := filepath.Join(s.backupDir(), record.Archive)
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return Record{}, err
	}
	if err = writeArchive(f, s.BaseDir, s.PanelDir, record); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return Record{}, err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(tmp)
		return Record{}, err
	}
	info, err := os.Stat(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return Record{}, err
	}
	record.SizeBytes = info.Size()
	if err = os.Rename(tmp, archivePath); err != nil {
		_ = os.Remove(tmp)
		return Record{}, err
	}
	if err = s.writeRecord(record); err != nil {
		_ = os.Remove(archivePath)
		return Record{}, err
	}
	if err = s.enforceRetention(maxVersions, record.ID); err != nil {
		return record, err
	}
	return record, nil
}

func writeArchive(out io.Writer, baseDir, panelDir string, record Record) error {
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	manifest, _ := json.MarshalIndent(record, "", "  ")
	if err := writeTarBytes(tw, "manifest.json", manifest, 0o600); err != nil {
		return err
	}
	for _, rel := range includedPaths {
		if err := addTree(tw, baseDir, rel); err != nil {
			return err
		}
	}
	for _, rel := range includedFiles {
		if _, err := os.Stat(filepath.Join(baseDir, rel)); err == nil {
			if err := addTree(tw, baseDir, rel); err != nil {
				return err
			}
		}
	}
	if record.PanelSettingsIncluded {
		if err := addExternalFile(tw, filepath.Join(panelDir, "panel_settings.json"), "PanelSettings/panel_settings.json"); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func addExternalFile(tw *tar.Writer, source, archiveName string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = archiveName
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

func addTree(tw *tar.Writer, baseDir, relRoot string) error {
	root := filepath.Join(baseDir, filepath.FromSlash(relRoot))
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		link := ""
		if info.Mode()&os.ModeSymlink != 0 {
			link, _ = os.Readlink(path)
		}
		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}
		header.Name = rel
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func writeTarBytes(tw *tar.Writer, name string, data []byte, mode int64) error {
	h := &tar.Header{Name: name, Mode: mode, Size: int64(len(data)), ModTime: time.Now()}
	if err := tw.WriteHeader(h); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func (s *Service) List() ([]Record, error) {
	entries, err := os.ReadDir(s.backupDir())
	if os.IsNotExist(err) {
		return []Record{}, nil
	}
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.backupDir(), entry.Name()))
		if err != nil {
			continue
		}
		var record Record
		if json.Unmarshal(data, &record) != nil || !validID(record.ID) {
			continue
		}
		if info, err := os.Stat(filepath.Join(s.backupDir(), record.Archive)); err == nil {
			record.SizeBytes = info.Size()
			records = append(records, record)
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].CreatedAt.After(records[j].CreatedAt) })
	return records, nil
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.readRecord(id)
	if err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(s.backupDir(), record.Archive)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Remove(filepath.Join(s.backupDir(), id+".json"))
}

func (s *Service) Restore(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.readRecord(id)
	if err != nil {
		return err
	}
	f, err := os.Open(filepath.Join(s.backupDir(), record.Archive))
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Clean(h.Name))
		if name == "manifest.json" || !allowedPath(name) {
			continue
		}
		dstBase := s.BaseDir
		relName := name
		if strings.HasPrefix(name, "PanelSettings/") {
			if s.PanelDir == "" {
				continue
			}
			dstBase = s.PanelDir
			relName = strings.TrimPrefix(name, "PanelSettings/")
		}
		dst := filepath.Join(dstBase, filepath.FromSlash(relName))
		if h.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
			continue
		}
		if h.Typeflag == tar.TypeSymlink || h.Typeflag == tar.TypeLink {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode)&0o777)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func (s *Service) readRecord(id string) (Record, error) {
	if !validID(id) {
		return Record{}, fmt.Errorf("invalid backup id")
	}
	data, err := os.ReadFile(filepath.Join(s.backupDir(), id+".json"))
	if err != nil {
		return Record{}, err
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil || record.ID != id || !validID(record.ID) {
		return Record{}, fmt.Errorf("invalid backup metadata")
	}
	return record, nil
}

func (s *Service) writeRecord(record Record) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.backupDir(), record.ID+".json"), data, 0o600)
}

func (s *Service) enforceRetention(maxVersions int, keepID string) error {
	if maxVersions <= 0 {
		maxVersions = 10
	}
	if maxVersions > 100 {
		maxVersions = 100
	}
	records, err := s.List()
	if err != nil {
		return err
	}
	if len(records) <= maxVersions {
		return nil
	}
	for _, record := range records[maxVersions:] {
		if record.ID == keepID {
			continue
		}
		_ = os.Remove(filepath.Join(s.backupDir(), record.Archive))
		_ = os.Remove(filepath.Join(s.backupDir(), record.ID+".json"))
	}
	return nil
}

func allowedPath(name string) bool {
	if name == "" || strings.HasPrefix(name, "../") || name == ".." || strings.Contains(name, "/../") {
		return false
	}
	for _, root := range append(includedPaths, includedFiles...) {
		if name == root || strings.HasPrefix(name, root+"/") {
			return true
		}
	}
	if name == "PanelSettings/panel_settings.json" {
		return true
	}
	return false
}

func validID(id string) bool {
	return len(id) > 20 && len(id) < 40 && !strings.ContainsAny(id, "/\\")
}

func sanitizeNote(note string) string {
	note = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(note, "\r", " "), "\n", " "))
	if len(note) > 200 {
		return note[:200]
	}
	return note
}
