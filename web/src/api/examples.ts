import { apiRequest } from "./client"

export type Example = {
  CreatedAt: string
  ID: string
  Name: string
}

export type CreateExampleRequest = {
  name: string
}

export function createExample(request: CreateExampleRequest) {
  return apiRequest<Example>("/example", {
    method: "POST",
    body: JSON.stringify(request),
  })
}
