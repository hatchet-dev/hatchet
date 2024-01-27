class HatchetPromise<T> {
  cancel: Function = (reason: any) => {};
  promise: Promise<T>;

  constructor(promise: Promise<T>) {
    this.promise = new Promise((resolve, reject) => {
      this.cancel = reject;
      Promise.resolve(promise).then(resolve).catch(reject);
    });
  }
}

export default HatchetPromise;
