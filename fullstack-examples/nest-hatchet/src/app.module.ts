import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { HatchetService } from './hatchet.service';

@Module({
  imports: [],
  controllers: [AppController],
  providers: [HatchetService],
})
export class AppModule {}
