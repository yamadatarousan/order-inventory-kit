import { FormEvent, useEffect, useState } from "react";
import {
  cancelOrder as cancelOrderApi,
  confirmPayment as confirmPaymentApi,
  fetchOrder as fetchOrderApi,
  type CancelOrderResult,
  type ConfirmPaymentInput,
  type ConfirmPaymentResult,
  type FetchOrderResult,
  type OrderView,
} from "./orderActionsApi";

type OrderActionsPanelProps = {
  fetchOrder?: (orderId: string) => Promise<FetchOrderResult>;
  cancelOrder?: (orderId: string) => Promise<CancelOrderResult>;
  confirmPayment?: (input: ConfirmPaymentInput) => Promise<ConfirmPaymentResult>;
  currentOrderId?: string;
};

export function OrderActionsPanel({
  fetchOrder = fetchOrderApi,
  cancelOrder = cancelOrderApi,
  confirmPayment = confirmPaymentApi,
  currentOrderId = "",
}: OrderActionsPanelProps) {
  const [amount, setAmount] = useState("");
  const [idempotencyKey, setIdempotencyKey] = useState("");
  const [message, setMessage] = useState("");
  const [order, setOrder] = useState<OrderView | null>(null);
  const targetOrderId = currentOrderId.trim();

  useEffect(() => {
    if (targetOrderId === "") {
      setOrder(null)
      return;
    }
    void loadOrder(targetOrderId);
  }, [targetOrderId]);

  async function loadOrder(orderId: string) {
    const result = await fetchOrder(orderId);
    if (result.kind === "success") {
      setOrder(result.order);
      return { ok: true as const };
    }
    if (result.kind === "notFound") {
      setOrder(null);
      return { ok: false as const, reason: "notFound" as const };
    }
    setOrder(null);
    return { ok: false as const, reason: "unknown" as const };
  }

  async function onFetchOrder() {
    setMessage("");
    if (targetOrderId === "") {
      setMessage("先に注文を作成してください");
      return;
    }
    const result = await loadOrder(targetOrderId);
    if (result.ok) {
      setMessage("注文を取得しました");
      return;
    }
    if (result.reason === "notFound") {
      setMessage("注文が見つかりません");
      return;
    }
    setMessage("注文取得に失敗しました");
  }

  async function onCancelOrder() {
    setMessage("");
    if (targetOrderId === "") {
      setMessage("先に注文を作成してください");
      return;
    }

    const result = await cancelOrder(targetOrderId);
    if (result.kind === "success") {
      setOrder(result.order);
      setMessage("注文をキャンセルしました");
      return;
    }
    if (result.kind === "notFound") {
      setMessage("注文が見つかりません");
      return;
    }
    setMessage("注文キャンセルに失敗しました");
  }

  async function onConfirmPayment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    if (targetOrderId === "") {
      setMessage("先に注文を作成してください");
      return;
    }

    const parsedAmount = Number(amount);
    if (
      idempotencyKey.trim() === "" ||
      amount.trim() === "" ||
      !Number.isInteger(parsedAmount) ||
      parsedAmount < 1
    ) {
      setMessage("入力内容を確認してください");
      return;
    }

    const result = await confirmPayment({
      orderId: targetOrderId,
      amount: parsedAmount,
      idempotencyKey: idempotencyKey.trim(),
    });
    if (result.kind === "success") {
      setMessage("決済を確定しました");
      return;
    }
    if (result.kind === "invalid") {
      setMessage("入力内容を確認してください");
      return;
    }
    if (result.kind === "notFound") {
      setMessage("注文が見つかりません");
      return;
    }
    if (result.kind === "amountConflict") {
      setMessage("金額が一致しません");
      return;
    }
    setMessage("決済確定に失敗しました");
  }

  return (
    <section className="panel panel-actions">
      <h2>注文操作</h2>
      {targetOrderId !== "" ? (
        <p className="hint-message">
          現在の注文ID: {targetOrderId}
        </p>
      ) : (
        <p className="hint-message">注文作成後に操作できます</p>
      )}

      <div className="actions-row">
        <button type="button" onClick={() => void onFetchOrder()}>
          注文を再取得
        </button>
        <button type="button" onClick={() => void onCancelOrder()}>
          注文をキャンセル
        </button>
      </div>

      <form onSubmit={onConfirmPayment} className="stack-form">
        <label htmlFor="amount">amount</label>
        <input
          id="amount"
          name="amount"
          type="number"
          inputMode="numeric"
          value={amount}
          onChange={(event) => setAmount(event.target.value)}
        />
        <label htmlFor="idempotencyKey">idempotencyKey</label>
        <input
          id="idempotencyKey"
          name="idempotencyKey"
          value={idempotencyKey}
          onChange={(event) => setIdempotencyKey(event.target.value)}
        />
        <button type="submit">決済を確定</button>
      </form>

      {message !== "" ? (
        <p role="status" className="status-message">
          {message}
        </p>
      ) : null}
      {order ? (
        <article className="result-card">
          <p>id: {order.id}</p>
          <p>customerId: {order.customerId}</p>
          <p>status: {order.status}</p>
          <ul>
            {order.items.map((item, index) => (
              <li key={`${item.sku}-${index}`}>
                {item.sku} x {item.quantity}
              </li>
            ))}
          </ul>
        </article>
      ) : null}
    </section>
  );
}
