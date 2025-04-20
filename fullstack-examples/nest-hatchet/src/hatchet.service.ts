import { Injectable } from '@nestjs/common';
import { HatchetClient } from '@hatchet-dev/typescript-sdk';
import { simple } from './tasks';

@Injectable()
export class HatchetService {
  private hatchet: HatchetClient;

  constructor() {
    this.hatchet = HatchetClient.init();
  }

  get simple() {
    return simple(this.hatchet);
  }
}
