export function isMapOfStringToStringArray(
  map: any
): map is Map<string, string[]> {
  if (!(map instanceof Map)) {
    return false;
  }

  for (const [key, value] of map.entries()) {
    if (typeof key !== "string") {
      return false;
    }

    if (
      !Array.isArray(value) ||
      !value.every((item) => typeof item === "string")
    ) {
      return false;
    }
  }

  return true;
}
