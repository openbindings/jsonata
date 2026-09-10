import type { JSONExecutor, JSONExecutorOptions } from './json-executor';
export interface NodeExecutorOptions extends JSONExecutorOptions {
  /** Wall deadline in milliseconds, from 1 through 2147483647. */
  readonly timeout?: number;
  readonly workers?: number;
  /** Total active plus queued evaluations, not a payload-byte allowance. */
  readonly maxPending?: number;
  /** V8 old-generation heap limit per worker; not a process RSS limit. */
  readonly maxWorkerHeapMB?: number;
}
export interface NodeExecutor extends JSONExecutor { close(): Promise<void> }
/** Optional Node 18+ pool; the owner must await close() at teardown. */
export function createNodeExecutor(options?: NodeExecutorOptions): NodeExecutor;
