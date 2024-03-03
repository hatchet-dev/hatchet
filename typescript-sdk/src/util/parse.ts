export function parseJSON(json: string): any {
  try {
    const firstParse = JSON.parse(json);

    // Hatchet engine versions <=0.14.0 return JSON as a quoted string which needs to be parsed again.
    // This is a workaround for that issue, but will not be needed in future versions.
    try {
      return JSON.parse(firstParse);
    } catch (e: any) {
      return firstParse;
    }
  } catch (e: any) {
    throw new Error(`Could not parse JSON: ${e.message}`);
  }
}
