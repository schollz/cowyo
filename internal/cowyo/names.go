package cowyo

import (
	cryptorand "crypto/rand"
	"errors"
	"math/big"
	"strings"
)

var adjectives = []string{
	"able", "active", "agile", "alert", "alive", "amazing", "amused", "ancient", "angelic", "angry", "anxious", "apt", "arctic", "artistic", "attentive", "attractive", "authentic", "autumnal", "awesome",
	"baby", "bashful", "basic", "beautiful", "beige", "beloved", "big", "bitter", "black", "blissful", "blue", "bold", "brave", "breezy", "bright", "brilliant", "broad", "bubbly", "busy",
	"calm", "candid", "careful", "charming", "cheerful", "chilly", "chubby", "clean", "clever", "cloudy", "clumsy", "coastal", "colorful", "comfy", "cool", "cosmic", "cozy", "crafty", "creamy", "crisp", "curious", "cute",
	"dainty", "dapper", "daring", "dark", "dazzling", "dear", "delicate", "delightful", "determined", "diligent", "dim", "dizzy", "dreamy", "dry", "dusty", "dynamic",
	"eager", "early", "earnest", "easy", "electric", "elegant", "emerald", "emotional", "empty", "endless", "energetic", "enormous", "epic", "equal", "essential", "ethical", "excited", "exotic", "expert",
	"fair", "faithful", "fancy", "fantastic", "fast", "fearless", "festive", "fiery", "fine", "firm", "fluffy", "foggy", "fond", "fortunate", "free", "fresh", "friendly", "frisky", "frosty", "funny", "fuzzy",
	"gentle", "giant", "gifted", "glad", "gleaming", "glossy", "golden", "good", "goofy", "graceful", "grand", "grateful", "gray", "great", "green", "groovy", "growing",
	"happy", "handsome", "hardy", "harmonious", "healthy", "helpful", "heroic", "honest", "hopeful", "hot", "huge", "humble", "hungry", "hushed",
	"icy", "ideal", "immense", "important", "impressive", "independent", "indigo", "innocent", "inventive", "ivory",
	"jaunty", "jazzy", "jolly", "joyful", "jubilant", "juicy",
	"keen", "kind", "kindly", "kingly", "knowing",
	"large", "late", "lazy", "light", "lively", "little", "lofty", "lonely", "long", "lovely", "loyal", "lucky", "luminous", "lush",
	"magical", "majestic", "major", "mellow", "merry", "mighty", "mild", "minty", "modern", "modest", "moody", "mossy", "musical", "muted", "mysterious",
	"narrow", "natural", "neat", "new", "nice", "nimble", "noble", "noisy", "normal", "notable", "novel", "nutty",
	"odd", "olive", "open", "optimistic", "orange", "orderly", "organic", "original", "outgoing", "oval",
	"pale", "patient", "peaceful", "peachy", "perfect", "perky", "pink", "playful", "pleasant", "plucky", "polite", "popular", "powerful", "precious", "pretty", "proud", "purple", "puzzled",
	"quaint", "qualified", "quick", "quiet", "quirky",
	"radiant", "rapid", "rare", "ready", "red", "regal", "relaxed", "reliable", "rich", "rosy", "rough", "round", "royal", "rural", "rustic",
	"safe", "sandy", "scarlet", "serene", "shaggy", "shiny", "short", "shy", "silent", "silky", "silly", "simple", "sleepy", "sleek", "slim", "slow", "small", "smart", "smooth", "snowy", "soft", "solid", "sparkling", "speedy", "spicy", "splendid", "spotted", "springy", "steady", "stormy", "strong", "sunny", "super", "sweet", "swift",
	"tall", "tame", "tangy", "tender", "thankful", "thoughtful", "tidy", "tiny", "tough", "tranquil", "tropical", "trustworthy", "turquoise",
	"uncommon", "unique", "united", "upbeat", "urban", "useful",
	"vast", "velvety", "vibrant", "violet", "vivid",
	"warm", "wary", "wavy", "wee", "whimsical", "white", "wild", "wise", "witty", "wonderful", "woolly",
	"yellow", "young", "youthful", "yummy",
	"zany", "zealous", "zesty", "zippy",
}

var animals = []string{
	"alligator", "alpaca", "ant", "anteater", "antelope", "ape", "armadillo",
	"baboon", "badger", "bat", "bear", "beaver", "bee", "beetle", "bird", "bison", "buffalo", "bull", "butterfly",
	"camel", "canary", "cat", "caterpillar", "cattle", "chameleon", "cheetah", "chicken", "chimpanzee", "clam", "cobra", "cod", "cougar", "cow", "coyote", "crab", "cricket", "crocodile", "crow",
	"deer", "dog", "dolphin", "donkey", "dove", "dragonfly", "duck",
	"eagle", "eel", "elephant", "elk", "emu",
	"falcon", "ferret", "finch", "fish", "flamingo", "flea", "fly", "fox", "frog",
	"gazelle", "gecko", "gerbil", "giraffe", "goat", "goose", "gorilla", "grasshopper", "guppy",
	"hamster", "hare", "hawk", "hedgehog", "hen", "hippo", "hornet", "horse", "hummingbird", "hyena",
	"ibis", "iguana", "insect",
	"jackal", "jaguar", "jay", "jellyfish",
	"kangaroo", "kitten", "kiwi", "koala",
	"ladybug", "lamb", "leopard", "lion", "lizard", "llama", "lobster",
	"manatee", "meerkat", "mink", "mole", "monkey", "moose", "mosquito", "moth", "mouse", "mule",
	"narwhal", "newt", "nightingale",
	"octopus", "opossum", "orangutan", "orca", "ostrich", "otter", "owl",
	"panda", "panther", "parrot", "peacock", "pelican", "penguin", "pig", "pigeon", "platypus", "pony", "porcupine", "pufferfish", "puffin", "python",
	"quail",
	"rabbit", "raccoon", "ram", "rat", "raven", "ray", "reindeer", "rhino", "robin", "rooster",
	"salamander", "salmon", "scorpion", "seal", "shark", "sheep", "skunk", "sloth", "snail", "snake", "sparrow", "spider", "squid", "squirrel", "starfish",
	"tadpole", "tarantula", "termite", "tiger", "toad", "tortoise", "toucan", "trout", "turkey", "turtle",
	"urchin",
	"viper", "vulture",
	"wallaby", "walrus", "wasp", "weasel", "whale", "wolf", "wombat", "woodpecker", "worm",
	"yak",
	"zebra",
}

var documentNames = alliterations(adjectives, animals)

func alliterations(adjectives, animals []string) []string {
	animalsByInitial := make(map[byte][]string)
	for _, animal := range animals {
		if initial, ok := slugInitial(animal); ok {
			animalsByInitial[initial] = append(animalsByInitial[initial], animal)
		}
	}

	var names []string
	for _, adjective := range adjectives {
		initial, ok := slugInitial(adjective)
		if !ok {
			continue
		}
		for _, animal := range animalsByInitial[initial] {
			names = append(names, adjective+"-"+animal)
		}
	}
	return names
}

func slugInitial(word string) (byte, bool) {
	word = strings.TrimSpace(word)
	if word == "" || word[0] < 'a' || word[0] > 'z' {
		return 0, false
	}
	for index := 1; index < len(word); index++ {
		if word[index] < 'a' || word[index] > 'z' {
			return 0, false
		}
	}
	return word[0], true
}

func randomDocumentName() (string, error) {
	if len(documentNames) == 0 {
		return "", errors.New("no alliterative document names configured")
	}

	index, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(len(documentNames))))
	if err != nil {
		return "", err
	}
	return documentNames[index.Int64()], nil
}
