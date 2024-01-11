export function getEnumKeyByValue(
  enumObject: any,
  enumValue: number
): string | undefined {
  return Object.keys(enumObject).find((key) => enumObject[key] === enumValue);
}
