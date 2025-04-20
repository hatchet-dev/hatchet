import { Injectable } from '@nestjs/common';
import { HatchetClient } from '@hatchet-dev/typescript-sdk';
import { hatchet } from './hatchet-client';
import { simple } from './tasks';

@Injectable()
export class HatchetService {
  private hatchet: HatchetClient;

  constructor() {
    this.hatchet = hatchet;
  }

  get simple() {
    return simple(this.hatchet);
  }
}
