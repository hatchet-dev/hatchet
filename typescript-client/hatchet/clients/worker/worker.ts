export class Worker {
  name: string;

  constructor(name: string) {
    this.name = name;
    throw new Error('not implemented');
  }

  //   static from_steps(steps) {
  //     const worker = new Worker();
  //     worker.steps = steps;
  //     return worker;
  //   }

  //   add_step(step) {
  //     this.steps.push(step);
  //   }

  //   run(input) {
  //     return next;
  //   }
}
