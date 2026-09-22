import { HttpService } from "@nestjs/axios"
import { Injectable } from "@nestjs/common"
import { firstValueFrom } from "rxjs"

type RemoteItem = { id: string; name: string }

@Injectable()
export class CatalogClient {
  constructor(private readonly http: HttpService) {}

  async findItem(id: string): Promise<RemoteItem> {
    const response = await firstValueFrom(this.http.get<RemoteItem>(`/items/${id}`))
    return response.data
  }
}
