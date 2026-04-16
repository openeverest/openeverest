// Type declarations for @marcbachmann/cel-js
declare module '@marcbachmann/cel-js' {
  export function evaluate(
    expression: string,
    context: Record<string, unknown>
  ): unknown;
}
