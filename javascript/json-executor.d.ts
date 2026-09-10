export interface NumericWorkLimits { readonly maxDigits: number; readonly maxExponent: number }
export interface JSONExecutorOptions {
  /** Milliseconds. Cooperative in-process; wall deadline in the Node pool. */
  readonly timeout?: number;
  readonly stack?: number;
  readonly sequence?: number;
  readonly cacheSize?: number;
  /** All length limits count UTF-16 code units in JSON/expression text. */
  readonly maxExpressionLength?: number;
  readonly maxInputLength?: number;
  readonly maxOutputLength?: number;
  readonly numericWork?: NumericWorkLimits;
}
/** Bindings use unprefixed variable names; duplicate object members are rejected. */
export interface EvaluationOptions { readonly bindingsJSON?: string; readonly signal?: AbortSignal }
export interface JSONExecutor {
  evaluate(expression: string, inputJSON: string, options?: EvaluationOptions): Promise<string>;
}
/** Closed text boundary; in-process cancellation is cooperative. */
export function createJSONataExecutor(options?: JSONExecutorOptions): JSONExecutor;
