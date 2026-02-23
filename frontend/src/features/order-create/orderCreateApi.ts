import { apiClient } from "../../api/client";

export type CreateOrderInput = {
  customerId: string;
  sku: string;
  quantity: number;
  unitPrice: number;
};

export type CreateOrderResult =
  | { kind: "success"; orderId: string; status: string }
  | { kind: "invalid" }
  | { kind: "priceConflict" }
  | { kind: "unknown" };

export async function createOrder(
  input: CreateOrderInput,
): Promise<CreateOrderResult> {
  try {
    const result = await apiClient.POST("/orders", {
      body: {
        customerId: input.customerId,
        items: [
          {
            sku: input.sku,
            quantity: input.quantity,
            unitPrice: input.unitPrice,
          },
        ],
      },
    });

    if (result.response.status === 200 && result.data) {
      return {
        kind: "success",
        orderId: result.data.orderId,
        status: result.data.status,
      };
    }

    if (result.response.status === 400) {
      return { kind: "invalid" };
    }

    if (result.response.status === 409) {
      return { kind: "priceConflict" };
    }

    return { kind: "unknown" };
  } catch {
    return { kind: "unknown" };
  }
}
