/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	// SlugLength is the length of generated project slugs
	SlugLength = 8
	// SlugCharset contains allowed characters for slug generation (lowercase alphanumeric)
	SlugCharset = "abcdefghijklmnopqrstuvwxyz0123456789"
	// RandomSuffixLength is the length of random characters appended to human-readable slugs
	RandomSuffixLength = 4
)

// Adjectives for human-readable slug generation
var adjectives = []string{
	// Nature & Seasons
	"autumn", "summer", "spring", "winter", "vernal", "estival", "autumnal", "hibernal",
	"alpine", "arctic", "tropical", "temperate", "polar", "equatorial", "seasonal", "verdant",

	// Colors
	"crimson", "scarlet", "vermillion", "burgundy", "ruby", "cherry", "rose", "pink",
	"orange", "amber", "golden", "yellow", "lemon", "saffron", "honey", "cream",
	"green", "emerald", "jade", "olive", "lime", "mint", "sage", "forest",
	"blue", "azure", "cobalt", "navy", "sapphire", "cerulean", "teal", "cyan",
	"purple", "violet", "lavender", "lilac", "plum", "mauve", "indigo", "amethyst",
	"white", "ivory", "pearl", "alabaster", "snow", "frost", "silver", "platinum",
	"black", "ebony", "onyx", "obsidian", "jet", "raven", "sable", "charcoal",
	"gray", "slate", "ash", "smoke", "mist", "fog", "cloud", "steel",
	"brown", "bronze", "copper", "brass", "rust", "mahogany", "chestnut", "umber",

	// Textures & Materials
	"silk", "velvet", "satin", "cotton", "linen", "wool", "cashmere", "fleece",
	"smooth", "rough", "coarse", "fine", "soft", "hard", "crisp", "delicate",
	"polished", "burnished", "lustrous", "glossy", "matte", "dull", "shiny", "bright",
	"crystal", "glass", "diamond", "quartz", "opal", "topaz", "beryl", "garnet",
	"stone", "marble", "granite", "slate", "limestone", "sandstone", "flint", "shale",
	"metal", "iron", "steel", "titanium", "aluminum", "zinc", "tin", "lead",

	// Sizes & Shapes
	"tiny", "small", "little", "petite", "miniature", "compact", "minute", "slight",
	"large", "big", "huge", "vast", "immense", "massive", "giant", "colossal",
	"long", "short", "tall", "high", "low", "deep", "shallow", "narrow",
	"wide", "broad", "thick", "thin", "slim", "slender", "lean", "stout",
	"round", "square", "flat", "curved", "bent", "straight", "angular", "circular",
	"steep", "gentle", "gradual", "sharp", "blunt", "pointed", "tapered", "jagged",

	// Weather & Time
	"misty", "foggy", "cloudy", "hazy", "murky", "dim", "dusky", "shadowy",
	"sunny", "bright", "clear", "brilliant", "radiant", "glowing", "luminous", "shining",
	"rainy", "stormy", "windy", "breezy", "gusty", "calm", "still", "tranquil",
	"snowy", "icy", "frosty", "frozen", "glacial", "frigid", "chilly", "cold",
	"warm", "hot", "scorching", "blazing", "torrid", "sweltering", "balmy", "mild",
	"dawn", "morning", "noon", "dusk", "twilight", "evening", "night", "midnight",

	// Qualities & States
	"hidden", "secret", "mysterious", "arcane", "cryptic", "enigmatic", "veiled", "obscure",
	"silent", "quiet", "hushed", "muted", "soundless", "noiseless", "still", "peaceful",
	"loud", "noisy", "boisterous", "clamorous", "thunderous", "deafening", "resounding", "vociferous",
	"swift", "quick", "rapid", "fast", "speedy", "hasty", "fleet", "brisk",
	"slow", "gradual", "leisurely", "languid", "sluggish", "dawdling", "lazy", "idle",
	"strong", "mighty", "powerful", "robust", "sturdy", "solid", "firm", "stable",
	"weak", "feeble", "fragile", "delicate", "frail", "brittle", "flimsy", "tender",

	// Emotions & Moods
	"happy", "joyful", "cheerful", "merry", "gleeful", "jolly", "jovial", "blissful",
	"sad", "melancholy", "somber", "gloomy", "morose", "doleful", "woeful", "plaintive",
	"calm", "serene", "placid", "tranquil", "peaceful", "composed", "unruffled", "balanced",
	"angry", "wrathful", "furious", "irate", "livid", "fierce", "savage", "wild",
	"proud", "noble", "regal", "majestic", "stately", "dignified", "august", "grand",
	"humble", "modest", "meek", "lowly", "simple", "plain", "unassuming", "demure",

	// Age & Condition
	"ancient", "old", "aged", "antique", "archaic", "primeval", "venerable", "timeworn",
	"young", "youthful", "juvenile", "fresh", "new", "novel", "recent", "modern",
	"pristine", "perfect", "flawless", "immaculate", "spotless", "pure", "clean", "unblemished",
	"worn", "weathered", "faded", "tattered", "ragged", "shabby", "decrepit", "dilapidated",
	"broken", "shattered", "fractured", "cracked", "split", "ruptured", "damaged", "ruined",
	"whole", "complete", "intact", "unbroken", "sound", "healthy", "robust", "vigorous",

	// Movement & Energy
	"flowing", "streaming", "rushing", "cascading", "pouring", "flooding", "surging", "gushing",
	"drifting", "floating", "soaring", "gliding", "sailing", "flying", "hovering", "wafting",
	"dancing", "swaying", "swinging", "rocking", "rolling", "tumbling", "spinning", "whirling",
	"falling", "dropping", "plunging", "diving", "descending", "sinking", "settling", "subsiding",
	"rising", "ascending", "climbing", "soaring", "mounting", "elevating", "lifting", "hoisting",
	"vibrant", "dynamic", "lively", "energetic", "vivacious", "spirited", "animated", "active",

	// Special Qualities
	"rare", "unique", "singular", "exceptional", "extraordinary", "remarkable", "uncommon", "unusual",
	"common", "ordinary", "typical", "regular", "normal", "standard", "conventional", "average",
	"holy", "sacred", "divine", "blessed", "hallowed", "consecrated", "sanctified", "revered",
	"wild", "untamed", "feral", "savage", "primitive", "primal", "natural", "organic",
	"tame", "domestic", "cultivated", "refined", "civilized", "cultured", "polished", "sophisticated",
	"free", "liberated", "unbound", "unfettered", "unconstrained", "unrestricted", "independent", "autonomous",
	"bound", "tied", "fastened", "secured", "attached", "linked", "connected", "joined",

	// Miscellaneous
	"nameless", "anonymous", "unknown", "mysterious", "enigmatic", "cryptic", "obscure", "hidden",
	"famous", "renowned", "celebrated", "acclaimed", "distinguished", "notable", "prominent", "eminent",
	"lucky", "fortunate", "blessed", "favored", "charmed", "auspicious", "propitious", "providential",
	"odd", "strange", "peculiar", "curious", "unusual", "bizarre", "eccentric", "quirky",
	"super", "supreme", "ultimate", "paramount", "preeminent", "transcendent", "sublime", "exalted",
	"royal", "regal", "imperial", "sovereign", "princely", "kingly", "queenly", "noble",
}

// Nouns for human-readable slug generation
var nouns = []string{
	// Water Bodies
	"ocean", "sea", "lake", "pond", "pool", "lagoon", "bay", "cove",
	"river", "stream", "creek", "brook", "tributary", "channel", "canal", "waterway",
	"waterfall", "cascade", "cataract", "rapids", "torrent", "fountain", "spring", "geyser",
	"wave", "ripple", "current", "tide", "surf", "swell", "breaker", "whitecap",
	"reef", "shoal", "sandbar", "atoll", "archipelago", "strait", "sound", "fjord",

	// Landforms
	"mountain", "peak", "summit", "ridge", "crest", "pinnacle", "precipice", "cliff",
	"hill", "knoll", "mound", "dune", "mesa", "butte", "plateau", "tableland",
	"valley", "gorge", "ravine", "canyon", "gulch", "gully", "dell", "hollow",
	"cave", "cavern", "grotto", "chamber", "tunnel", "passage", "vault", "den",
	"island", "isle", "islet", "atoll", "key", "peninsula", "cape", "headland",
	"desert", "dune", "oasis", "mirage", "wasteland", "badlands", "steppe", "plain",
	"field", "meadow", "prairie", "grassland", "pasture", "lea", "glade", "clearing",

	// Vegetation
	"forest", "woods", "woodland", "grove", "thicket", "copse", "brake", "spinney",
	"tree", "oak", "pine", "maple", "birch", "willow", "elm", "ash",
	"cedar", "cypress", "redwood", "sequoia", "mahogany", "teak", "bamboo", "palm",
	"flower", "blossom", "bloom", "petal", "bud", "rose", "lily", "orchid",
	"daisy", "tulip", "sunflower", "violet", "iris", "jasmine", "lavender", "poppy",
	"grass", "reed", "rush", "sedge", "fern", "moss", "lichen", "algae",
	"bush", "shrub", "hedge", "bramble", "thorn", "vine", "ivy", "creeper",
	"garden", "orchard", "vineyard", "plantation", "nursery", "arboretum", "conservatory", "greenhouse",

	// Weather & Sky
	"sky", "heaven", "firmament", "ether", "atmosphere", "stratosphere", "troposphere", "ionosphere",
	"cloud", "cumulus", "cirrus", "stratus", "nimbus", "fog", "mist", "haze",
	"rain", "drizzle", "shower", "downpour", "deluge", "monsoon", "precipitation", "rainfall",
	"snow", "snowflake", "blizzard", "flurry", "sleet", "hail", "frost", "ice",
	"wind", "breeze", "gust", "gale", "tempest", "storm", "hurricane", "typhoon",
	"thunder", "lightning", "bolt", "flash", "spark", "flare", "blaze", "glare",
	"sun", "sunlight", "sunshine", "sunbeam", "ray", "beam", "radiance", "glow",
	"moon", "moonlight", "moonbeam", "crescent", "gibbous", "lunar", "satellite", "orb",

	// Celestial
	"star", "stellar", "constellation", "asterism", "cluster", "supernova", "pulsar", "quasar",
	"planet", "mercury", "venus", "mars", "jupiter", "saturn", "uranus", "neptune",
	"comet", "meteor", "asteroid", "meteorite", "bolide", "fireball", "shooting", "falling",
	"galaxy", "nebula", "cosmos", "universe", "void", "expanse", "infinity", "eternity",
	"aurora", "borealis", "australis", "lights", "glow", "shimmer", "gleam", "glimmer",
	"eclipse", "solstice", "equinox", "zenith", "nadir", "meridian", "horizon", "twilight",

	// Animals & Birds
	"bird", "eagle", "hawk", "falcon", "raven", "crow", "owl", "dove",
	"swan", "heron", "crane", "stork", "pelican", "seagull", "albatross", "penguin",
	"butterfly", "dragonfly", "firefly", "beetle", "ladybug", "moth", "cricket", "cicada",
	"deer", "elk", "moose", "caribou", "antelope", "gazelle", "buffalo", "bison",
	"wolf", "fox", "bear", "lion", "tiger", "leopard", "panther", "lynx",
	"whale", "dolphin", "seal", "otter", "walrus", "manatee", "dugong", "porpoise",
	"fish", "salmon", "trout", "bass", "pike", "sturgeon", "marlin", "tuna",
	"frog", "toad", "newt", "salamander", "lizard", "gecko", "iguana", "chameleon",

	// Time & Moments
	"dawn", "sunrise", "daybreak", "morning", "forenoon", "noon", "midday", "afternoon",
	"dusk", "sunset", "twilight", "evening", "nightfall", "night", "midnight", "witching",
	"moment", "instant", "second", "minute", "hour", "day", "week", "season",
	"spring", "summer", "autumn", "winter", "solstice", "equinox", "harvest", "thaw",

	// Abstract Concepts
	"dream", "vision", "fantasy", "reverie", "daydream", "nightmare", "illusion", "mirage",
	"shadow", "shade", "silhouette", "outline", "form", "shape", "figure", "profile",
	"light", "radiance", "brilliance", "luminosity", "incandescence", "phosphorescence", "fluorescence", "bioluminescence",
	"darkness", "gloom", "murk", "dimness", "obscurity", "shadow", "shade", "umbra",
	"silence", "quiet", "hush", "stillness", "calm", "peace", "serenity", "tranquility",
	"sound", "noise", "din", "clamor", "uproar", "tumult", "cacophony", "symphony",
	"voice", "whisper", "murmur", "utterance", "speech", "word", "echo", "resonance",

	// Elements & Materials
	"fire", "flame", "blaze", "inferno", "pyre", "ember", "spark", "cinder",
	"earth", "soil", "clay", "loam", "sand", "gravel", "pebble", "boulder",
	"stone", "rock", "flint", "granite", "marble", "slate", "limestone", "quartz",
	"metal", "iron", "steel", "copper", "bronze", "silver", "gold", "platinum",
	"gem", "jewel", "crystal", "diamond", "ruby", "sapphire", "emerald", "topaz",
	"pearl", "opal", "jade", "amber", "coral", "ivory", "obsidian", "onyx",

	// Structures & Features
	"bridge", "arch", "span", "viaduct", "aqueduct", "causeway", "overpass", "trestle",
	"tower", "spire", "steeple", "minaret", "turret", "belfry", "campanile", "obelisk",
	"gate", "portal", "doorway", "entrance", "threshold", "passage", "corridor", "hallway",
	"wall", "rampart", "bulwark", "bastion", "parapet", "battlement", "fortification", "barrier",
	"path", "trail", "track", "route", "road", "highway", "byway", "lane",
	"shore", "coast", "beach", "strand", "waterfront", "shoreline", "seashore", "seaside",
	"marsh", "swamp", "bog", "fen", "wetland", "mire", "quagmire", "morass",
}

// GenerateRandomSlug generates a random 8-character lowercase alphanumeric slug
func GenerateRandomSlug() (string, error) {
	slug := make([]byte, SlugLength)
	charsetLength := big.NewInt(int64(len(SlugCharset)))

	for i := 0; i < SlugLength; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		slug[i] = SlugCharset[randomIndex.Int64()]
	}

	return string(slug), nil
}

// GenerateHumanReadableSlug generates a human-readable slug
// Format: <adjective>-<noun>-<random-chars>
// Example: copper-forest-7x9k, silver-lake-5k3x
func GenerateHumanReadableSlug() (string, error) {
	// Pick random adjective
	adjIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(adjectives))))
	if err != nil {
		return "", err
	}
	adjective := adjectives[adjIndex.Int64()]

	// Pick random noun
	nounIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(nouns))))
	if err != nil {
		return "", err
	}
	noun := nouns[nounIndex.Int64()]

	// Generate random suffix
	suffix := make([]byte, RandomSuffixLength)
	charsetLength := big.NewInt(int64(len(SlugCharset)))
	for i := 0; i < RandomSuffixLength; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		suffix[i] = SlugCharset[randomIndex.Int64()]
	}

	return fmt.Sprintf("%s-%s-%s", adjective, noun, string(suffix)), nil
}
