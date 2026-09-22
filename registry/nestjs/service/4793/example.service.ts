import { Injectable, NotFoundException } from "@nestjs/common"
import { PrismaService } from "../../infrastructure/prisma/prisma.service"
import { CreateItemDto } from "./dto/create-item.dto"

@Injectable()
export class ItemsService {
  constructor(private readonly prisma: PrismaService) {}

  async findOne(id: string) {
    const item = await this.prisma.item.findUnique({ where: { id } })
    if (!item) throw new NotFoundException("Item not found")
    return item
  }

  create(input: CreateItemDto) {
    return this.prisma.item.create({ data: input })
  }
}
