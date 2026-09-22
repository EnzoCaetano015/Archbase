import { Body, Controller, Get, Param, Post } from "@nestjs/common"
import { CreateItemDto } from "./dto/create-item.dto"
import { ItemsService } from "./items.service"

@Controller("items")
export class ItemsController {
  constructor(private readonly itemsService: ItemsService) {}

  @Get(":id")
  findOne(@Param("id") id: string) {
    return this.itemsService.findOne(id)
  }

  @Post()
  create(@Body() input: CreateItemDto) {
    return this.itemsService.create(input)
  }
}
