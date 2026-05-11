/**
 * Parses a Mabinogi metadata string into a key-value record.
 * Format: KEY:TYPE:VALUE;KEY:TYPE:VALUE;...
 * Example: MCAGT:8:63913845591482;MCMBAC:f:42.618401;MCMBAMAX:f:80.97496;MCMBAMIN:f:80.97496;MCMBGA:b:true;
 */
export function parseMabinogiMetadata(metadata: string): Record<string, any> {
  const result: Record<string, any> = {};
  if (!metadata) return result;

  const pairs = metadata.split(";");
  for (const pair of pairs) {
    if (!pair.trim()) continue;

    const parts = pair.split(":");
    if (parts.length < 3) continue;

    const key = parts[0];
    const type = parts[1];
    const rawValue = parts.slice(2).join(":"); // Handle cases where value might contain colons

    let value: any = rawValue;
    switch (type) {
      case "f":
        value = parseFloat(rawValue);
        break;
      case "b":
        value = rawValue.toLowerCase() === "true";
        break;
      case "8":
      case "i":
      case "d":
        value = parseInt(rawValue, 10);
        break;
      // Add more types if needed
    }
    result[key] = value;
  }

  return result;
}
