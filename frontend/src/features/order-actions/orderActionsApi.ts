import { apiClient } from "../../api/client";

export type OrderItemView = {
  sku: string;
  quantity: number;
};

export type OrderView = {
  id: string;
  customerId: string;
  status: string;
  items: OrderItemView[];
};

export type FetchOrderResult =
  | { kind: "success"; order: OrderView }
  | { kind: "notFound" }
  | { kind: "unknown" };

export type CancelOrderResult =
  | { kind: "success"; order: OrderView }
  | { kind: "notFound" }
  | { kind: "unknown" };

export type ConfirmPaymentInput = {
  orderId: string;
  amount: number;
  idempotencyKey: string;
};

export type ConfirmPaymentResult =
  | { kind: "success"; orderId: string; paymentStatus: string }
  | { kind: "invalid" }
  | { kind: "notFound" }
  | { kind: "amountConflict" }
  | { kind: "unknown" };

function toOrderView(data: {
  id?: string;
  customerId?: string;
  status?: string;
  items?: { sku?: string; quantity?: number }[];
}): OrderView {
  return {
    id: data.id ?? "",
    customerId: data.customerId ?? "",
    status: data.status ?? "",
    items: (data.items ?? []).map((item) => ({
      sku: item.sku ?? "",
      quantity: item.quantity ?? 0,
    })),
  };
}

export async function fetchOrder(orderId: string): Promise<FetchOrderResult> {
  try {
    const result = await apiClient.GET("/orders/{orderId}", {
      params: { path: { orderId } },
    });
    if (result.response.status === 200 && result.data) {
      return { kind: "success", order: toOrderView(result.data) };
    }
    if (result.response.status === 404) {
      return { kind: "notFound" };
    }
    return { kind: "unknown" };
  } catch {
    return { kind: "unknown" };
  }
}

export async function cancelOrder(orderId: string): Promise<CancelOrderResult> {
  try {
    const result = await apiClient.POST("/orders/{orderId}/cancel", {
      params: { path: { orderId } },
    });
    if (result.response.status === 200 && result.data) {
      return { kind: "success", order: toOrderView(result.data) };
    }
    if (result.response.status === 404) {
      return { kind: "notFound" };
    }
    return { kind: "unknown" };
  } catch {
    return { kind: "unknown" };
  }
}

export async function confirmPayment(
  input: ConfirmPaymentInput,
): Promise<ConfirmPaymentResult> {
  try {
    const result = await apiClient.POST("/payments/confirm", {
      body: {
        orderId: input.orderId,
        amount: input.amount,
        idempotencyKey: input.idempotencyKey,
      },
    });
    if (result.response.status === 200 && result.data) {
      return {
        kind: "success",
        orderId: result.data.orderId,
        paymentStatus: result.data.paymentStatus,
      };
    }
    if (result.response.status === 400) {
      return { kind: "invalid" };
    }
    if (result.response.status === 404) {
      return { kind: "notFound" };
    }
    if (result.response.status === 409) {
      return { kind: "amountConflict" };
    }
    return { kind: "unknown" };
  } catch {
    return { kind: "unknown" };
  }
}
