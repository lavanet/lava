import { StringToArrayMap } from "../common/common";

export function isStringToArrayMap(obj: any): obj is StringToArrayMap {
  // Check if obj is an object
  if (typeof obj !== "object" || obj === null) {
    return false;
  }

  // Check each property in the object
  for (const key in obj) {
    if (obj.hasOwnProperty(key)) {
      // Check if the value associated with each key is an array of strings
      if (
        !Array.isArray(obj[key]) ||
        !obj[key].every((item: any) => typeof item === "string")
      ) {
        return false;
      }
    }
  }
  return true;
}
