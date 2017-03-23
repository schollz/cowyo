adjectives = {}
with open('adjectives','r') as f:
    for line in f:
        word = line.strip().lower()
        if word[0] not in adjectives:
            adjectives[word[0]] = []
        adjectives[word[0]].append(word)

print(len(adjectives.keys()))

animals = {}
for aword in open('animals','r').read().split(','):
    word = aword.strip().lower()
    if word[0] not in animals:
        animals[word[0]] = []
    animals[word[0]].append(word)

print(len(animals))

i = 0
for key in adjectives.keys():
    if key in animals and key in adjectives:
        i = i + len(adjectives[key])*len(animals[key])

print(i)
