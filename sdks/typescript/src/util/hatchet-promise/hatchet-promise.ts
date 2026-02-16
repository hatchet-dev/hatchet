class HatchetPromise<T> {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  cancel: (reason?: any) => void = (_reason?: any) => {};
  promise: Promise<T>;
  /**
   * The original (non-cancelable) promise passed to the constructor.
   *
   * `promise` is a cancelable wrapper which rejects immediately when `cancel` is called.
   * `inner` continues executing and will settle when the underlying work completes.
   */
  inner: Promise<T>;

  constructor(promise: Promise<T>) {
    this.inner = Promise.resolve(promise) as Promise<T>;
    this.promise = new Promise((resolve, reject) => {
      this.cancel = reject;
      this.inner.then(resolve).catch(reject);
    });
  }
}

export default HatchetPromise;
