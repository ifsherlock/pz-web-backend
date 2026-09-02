package config

// These aliases come from Build 42 SandboxOptions.setTranslation calls.
var sandboxTranslationAliases = map[string]string{
	"Zombies": "ZombieCount", "Distribution": "ZombieDistribution",
	"WaterShutModifier": "WaterShut", "ElecShutModifier": "ElecShut", "AlarmDecayModifier": "AlarmDecay",
	"Temperature": "WorldTemperature", "Rain": "RainAmount", "Farming": "FarmingSpeed",
	"StatsDecrease": "StatDecrease", "NatureAbundance": "NatureAmount", "Alarm": "HouseAlarmFrequency",
	"LockedHouses": "LockedHouseFrequency", "FoodRotSpeed": "FoodSpoil", "FridgeFactor": "FridgeEffect",
	"EndRegen": "EnduranceRegen", "CarAlarm": "CarAlarmFrequency", "FishAbundance": "FishAmount",
	"ZombieLore.Speed": "ZSpeed", "ZombieLore.SprinterPercentage": "ZSprinterPercentage",
	"ZombieLore.Strength": "ZStrength", "ZombieLore.Toughness": "ZToughness",
	"ZombieLore.Transmission": "ZTransmission", "ZombieLore.Mortality": "ZInfectionMortality",
	"ZombieLore.Reanimate": "ZReanimateTime", "ZombieLore.Cognition": "ZCognition",
	"ZombieLore.DoorOpeningPercentage": "ZDoorOpeningPercentage", "ZombieLore.CrawlUnderVehicle": "ZCrawlUnderVehicle",
	"ZombieLore.Memory": "ZMemory", "ZombieLore.Sight": "ZSight", "ZombieLore.Hearing": "ZHearing",
	"ZombieLore.PlayerSpawnZombieRemoval": "ZSpawnRemoval",
}

// These aliases come from Build 42 SandboxOptions.setValueTranslation calls
// and from enum options whose setTranslation name also owns the option labels.
var sandboxValueTranslationAliases = map[string]string{
	"DayNightCycle": "DayNightCycle", "ClimateCycle": "ClimateCycle", "FogCycle": "FogCycle",
	"WaterShut": "Shutoff", "AlarmDecay": "Shutoff", "PlantAbundance": "FarmingAmount",
	"Helicopter": "HelicopterFreq", "MetaEvent": "MetaEventFreq", "SleepingEvent": "MetaEventFreq",
	"VehicleStoryChance": "SurvivorHouseChance", "ZoneStoryChance": "SurvivorHouseChance",
	"Temperature": "WorldTemperature", "Rain": "RainAmount", "StatsDecrease": "StatDecrease",
	"NatureAbundance": "NatureAmount", "Alarm": "HouseAlarmFrequency", "LockedHouses": "LockedHouseFrequency",
	"FoodRotSpeed": "FoodSpoil", "FridgeFactor": "FridgeEffect", "EndRegen": "EnduranceRegen",
	"CarAlarm": "CarAlarmFrequency", "FishAbundance": "FishAmount",
	"AnimalStatsModifier": "AnimalSpeed", "AnimalMetaStatsModifier": "AnimalSpeed",
	"AnimalPregnancyTime": "AnimalSpeed", "AnimalAgeModifier": "AnimalSpeed",
	"AnimalMilkIncModifier": "AnimalSpeed", "AnimalWoolIncModifier": "AnimalSpeed",
	"AnimalEggHatch": "AnimalSpeed", "AnimalRanchChance": "AnimalRanchChance",
	"AnimalTrackChance": "HouseAlarmFrequency", "AnimalPathChance": "HouseAlarmFrequency",
	"ZombieLore.Speed": "ZSpeed", "ZombieLore.Strength": "ZStrength", "ZombieLore.Toughness": "ZToughness",
	"ZombieLore.Transmission": "ZTransmission", "ZombieLore.Mortality": "ZInfectionMortality",
	"ZombieLore.Reanimate": "ZReanimateTime", "ZombieLore.Cognition": "ZCognition",
	"ZombieLore.CrawlUnderVehicle": "ZCrawlUnderVehicle", "ZombieLore.Memory": "ZMemory",
	"ZombieLore.Sight": "ZSight", "ZombieLore.Hearing": "ZHearing",
	"ZombieLore.PlayerSpawnZombieRemoval": "ZSpawnRemoval",
}
