// A quoted string is not a path: a map key like 'example.com/role' would
// otherwise be watched as a form field.
const STRING_LITERAL = /'(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"/g;

export const extractCelFieldPaths = (celExpr: string): string[][] => {
  // Regex matches patterns like: word.word.word (at least 2 segments)
  const regex = /([a-zA-Z0-9_]+(?:\.[a-zA-Z0-9_]+)+)/g;
  const matches = celExpr.replace(STRING_LITERAL, "''").match(regex) || [];

  // Remove duplicates and split each path into an array
  return Array.from(new Set(matches)).map((field) => field.split('.'));
};
