export type MockProduct = {
  sku: string;
  name: string;
  unitPrice: number;
  currency: "JPY";
  isActive: boolean;
};

export type MockCartItem = {
  sku: string;
  name: string;
  quantity: number;
  unitPrice: number;
  lineAmount: number;
};

export type MockCartTotals = {
  subtotal: number;
  shippingFee: number;
  serviceFee: number;
  total: number;
};

export type MockOrderSummary = {
  orderId: string;
  status: "accepted" | "confirmed" | "canceled";
  createdAt: string;
  totalAmount: number;
};

export type MockOrderDetail = {
  orderId: string;
  customerId: string;
  status: "accepted" | "confirmed" | "canceled";
  items: MockCartItem[];
  subtotal: number;
  shippingFee: number;
  serviceFee: number;
  totalAmount: number;
};

const MOCK_PRODUCTS: MockProduct[] = [
  { sku: "sku-1", name: "商品A", unitPrice: 100, currency: "JPY", isActive: true },
  { sku: "sku-2", name: "商品B", unitPrice: 250, currency: "JPY", isActive: true },
  { sku: "sku-stop", name: "販売停止商品", unitPrice: 300, currency: "JPY", isActive: false },
];

const MOCK_ORDER_DETAILS: MockOrderDetail[] = [
  {
    orderId: "hist-1",
    customerId: "c-1",
    status: "confirmed",
    items: [
      { sku: "sku-1", name: "商品A", quantity: 1, unitPrice: 100, lineAmount: 100 },
      { sku: "sku-2", name: "商品B", quantity: 2, unitPrice: 250, lineAmount: 500 },
    ],
    subtotal: 600,
    shippingFee: 500,
    serviceFee: 100,
    totalAmount: 1200,
  },
  {
    orderId: "hist-2",
    customerId: "c-1",
    status: "accepted",
    items: [{ sku: "sku-2", name: "商品B", quantity: 1, unitPrice: 250, lineAmount: 250 }],
    subtotal: 250,
    shippingFee: 500,
    serviceFee: 100,
    totalAmount: 850,
  },
  {
    orderId: "hist-3",
    customerId: "c-2",
    status: "canceled",
    items: [{ sku: "sku-1", name: "商品A", quantity: 2, unitPrice: 100, lineAmount: 200 }],
    subtotal: 200,
    shippingFee: 0,
    serviceFee: 0,
    totalAmount: 200,
  },
];

function toDelay<T>(value: T): Promise<T> {
  return Promise.resolve(value);
}

export async function listProducts(): Promise<MockProduct[]> {
  return toDelay(MOCK_PRODUCTS.map((p) => ({ ...p })));
}

export async function getProductDetail(
  sku: string,
): Promise<{ kind: "success"; product: MockProduct } | { kind: "notFound" }> {
  const product = MOCK_PRODUCTS.find((p) => p.sku === sku);
  if (!product) {
    return toDelay({ kind: "notFound" });
  }
  return toDelay({ kind: "success", product: { ...product } });
}

export function calculateCartTotals(items: MockCartItem[]): MockCartTotals {
  const subtotal = items.reduce((acc, item) => acc + item.lineAmount, 0);
  const shippingFee = subtotal > 0 ? 500 : 0;
  const serviceFee = subtotal > 0 ? 100 : 0;
  return {
    subtotal,
    shippingFee,
    serviceFee,
    total: subtotal + shippingFee + serviceFee,
  };
}

export function resolveProductForCart(sku: string): MockProduct | null {
  const product = MOCK_PRODUCTS.find((p) => p.sku === sku);
  if (!product || !product.isActive) {
    return null;
  }
  return product;
}

export async function listHistory(
  customerId: string,
): Promise<MockOrderSummary[]> {
  const summaries = MOCK_ORDER_DETAILS.filter((order) => order.customerId === customerId).map(
    (order) => ({
      orderId: order.orderId,
      status: order.status,
      createdAt:
        order.orderId === "hist-1"
          ? "2026-02-20T10:00:00Z"
          : order.orderId === "hist-2"
            ? "2026-02-21T11:00:00Z"
            : "2026-02-22T12:00:00Z",
      totalAmount: order.totalAmount,
    }),
  );
  return toDelay(summaries);
}

export async function getHistoryDetail(
  customerId: string,
  orderId: string,
): Promise<{ kind: "success"; order: MockOrderDetail } | { kind: "notFound" }> {
  const order = MOCK_ORDER_DETAILS.find(
    (candidate) => candidate.customerId === customerId && candidate.orderId === orderId,
  );
  if (!order) {
    return toDelay({ kind: "notFound" });
  }
  return toDelay({
    kind: "success",
    order: {
      ...order,
      items: order.items.map((item) => ({ ...item })),
    },
  });
}
