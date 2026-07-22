// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const isPromise = <T = unknown>(value: any): value is Promise<T> =>
  !!value &&
  typeof value === 'object' &&
  typeof value.then === 'function' &&
  typeof value.catch === 'function'
