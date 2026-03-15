export default <T extends { expires: string }>(array: T[]): T[] => {
  return array.sort((a, b) => {
    return a.expires.localeCompare(b.expires);
  });
};
