// List of words for generating random tenant names
export const adjectives = [
  'swift',
  'rapid',
  'quick',
  'fast',
  'agile',
  'nimble',
  'speedy',
  'fleet',
  'brisk',
  'prompt',
  'hasty',
  'sudden',
  'active',
  'lively',
  'dynamic',
  'busy',
  'eager',
  'fierce',
  'bold',
  'brave',
  'calm',
  'clever',
  'cool',
  'cozy',
  'gentle',
  'happy',
  'keen',
  'kind',
  'loyal',
  'lucky',
  'neat',
  'nice',
  'proud',
  'smart',
  'wise',
  'fresh',
  'green',
  'blue',
  'red',
  'gold',
];

export const nouns = [
  'hatchet',
  'runner',
  'worker',
  'server',
  'agent',
  'system',
  'engine',
  'flow',
  'stream',
  'river',
  'cloud',
  'sky',
  'star',
  'moon',
  'sun',
  'planet',
  'galaxy',
  'cosmos',
  'atom',
  'byte',
  'bit',
  'data',
  'code',
  'app',
  'stack',
  'queue',
  'array',
  'node',
  'tree',
  'graph',
  'web',
  'net',
  'hub',
  'core',
  'base',
  'unit',
  'cell',
  'team',
  'crew',
  'squad',
];

export const animals = [
  'fox',
  'wolf',
  'bear',
  'eagle',
  'hawk',
  'tiger',
  'lion',
  'puma',
  'lynx',
  'deer',
  'moose',
  'elk',
  'bison',
  'otter',
  'seal',
  'shark',
  'whale',
  'dolphin',
  'crow',
  'raven',
  'owl',
  'falcon',
  'robin',
  'finch',
  'crane',
  'heron',
  'swan',
  'goose',
  'duck',
  'turtle',
  'snake',
  'viper',
  'cobra',
  'gecko',
  'lizard',
  'iguana',
  'frog',
  'toad',
  'newt',
  'cat',
];

export const locations = [
  'peak',
  'ridge',
  'valley',
  'canyon',
  'forest',
  'jungle',
  'desert',
  'tundra',
  'mountain',
  'hill',
  'lake',
  'river',
  'ocean',
  'sea',
  'bay',
  'coast',
  'shore',
  'island',
  'plateau',
  'mesa',
  'glacier',
  'cave',
  'basin',
  'dune',
  'reef',
  'meadow',
  'plain',
  'prairie',
  'swamp',
  'marsh',
  'spring',
  'delta',
  'creek',
  'brook',
  'lagoon',
  'harbor',
  'port',
  'cove',
  'field',
  'garden',
];

const getRandomItem = <T>(array: T[]): T => {
  return array[Math.floor(Math.random() * array.length)];
};

/**
 * Generates a random name suitable for tenants, using combinations of words
 * @returns A hyphen-separated string of random words
 */
export function generateRandomName(): string {
  // Randomly decide which combination pattern to use (8 total patterns)
  const pattern = Math.floor(Math.random() * 7);

  switch (pattern) {
    case 0:
      // adjective-noun
      return `${getRandomItem(adjectives)}-${getRandomItem(nouns)}`;
    case 1:
      // adjective-animal
      return `${getRandomItem(adjectives)}-${getRandomItem(animals)}`;
    case 2:
      // animal-location
      return `${getRandomItem(animals)}-${getRandomItem(locations)}`;
    case 3:
      // noun-location
      return `${getRandomItem(nouns)}-${getRandomItem(locations)}`;
    case 5:
      // adjective-animal-location
      return `${getRandomItem(adjectives)}-${getRandomItem(animals)}-${getRandomItem(locations)}`;
    case 6:
      // animal-noun-location
      return `${getRandomItem(animals)}-${getRandomItem(nouns)}-${getRandomItem(locations)}`;
    default:
      // Fallback to adjective-noun
      return `${getRandomItem(adjectives)}-${getRandomItem(nouns)}`;
  }
}

/**
 * Validates if a name follows tenant name requirements
 * @param value The name to validate
 * @returns True if valid, or an error message string if invalid
 */
export function validateTenantName(value: string): true | string {
  const regex = /^[a-z0-9-]+$/;
  if (!value) {
    return 'Name is required';
  }
  if (!regex.test(value)) {
    return 'Name must be lowercase with only letters, numbers, and hyphens';
  }
  return true;
}
