import { Controller, Get } from '@nestjs/common';
import { AppService } from './app.service';
import { simple } from './tasks/simple';

@Controller()
export class AppController {
  constructor(private readonly appService: AppService) {}

  @Get()
  async getHello(): Promise<string> {
    const result = await simple.run({ Message: 'Hello, world!' });
    return result.TransformedMessage;
  }
}
