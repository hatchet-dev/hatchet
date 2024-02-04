class HatchetError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'HatchetError';
  }
}

export default HatchetError;
