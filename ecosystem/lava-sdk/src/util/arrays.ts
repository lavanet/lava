export function shuffleArray<T>(arr: T[]): T[] {
  const newArr = arr.slice();

  // Fisher-Yates shuffle
  for (let i = arr.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [newArr[i], newArr[j]] = [newArr[j], newArr[i]];
  }

  return arr;
}

export function isStringArray(array: any): array is Array<string> {
  if (!(array instanceof Array) || !Array.isArray(array)) {
    return false;
  }

  for (const value of array) {
    if (typeof value !== "string") {
      return false;
    }
  }

  return true;
}
