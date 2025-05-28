import { Controller, Get } from '@nestjs/common';
import { HatchetService } from './hatchet.service';

@Controller()
export class AppController {
  constructor(private readonly hatchet: HatchetService) {}

  @Get()
  async getHello(): Promise<string> {
    const result = await this.hatchet.simple.run({ Message: 'Hello, world!' });
    return result.TransformedMessage;
  }
}
