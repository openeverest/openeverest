import { CEL_SELF_VARIABLE } from './cel-validation.constants';

export const extractCelFieldPaths = (celExpr: string): string[][] => {
  // Regex matches patterns like: word.word.word (at least 2 segments)
  const regex = /([a-zA-Z0-9_]+(?:\.[a-zA-Z0-9_]+)+)/g;
  const matches = celExpr.match(regex) || [];

  // Remove duplicates and split each path into an array
  return Array.from(new Set(matches))
    .map((field) => field.split('.'))
    .filter(([root]) => root !== CEL_SELF_VARIABLE);
};
