import { ApiProperty } from "@nestjs/swagger"
import { IsNotEmpty, IsString, MaxLength } from "class-validator"

export class CreateItemDto {
  @ApiProperty({ example: "Example item" })
  @IsString()
  @IsNotEmpty()
  @MaxLength(100)
  name!: string
}
