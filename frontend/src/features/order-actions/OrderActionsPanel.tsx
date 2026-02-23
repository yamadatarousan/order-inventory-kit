import { FormEvent, useState } from "react";
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
};

export function OrderActionsPanel({
  fetchOrder = fetchOrderApi,
  cancelOrder = cancelOrderApi,
  confirmPayment = confirmPaymentApi,
}: OrderActionsPanelProps) {
  const [lookupOrderId, setLookupOrderId] = useState("");
  const [cancelOrderId, setCancelOrderId] = useState("");
  const [paymentOrderId, setPaymentOrderId] = useState("");
  const [amount, setAmount] = useState("");
  const [idempotencyKey, setIdempotencyKey] = useState("");
  const [message, setMessage] = useState("");
  const [order, setOrder] = useState<OrderView | null>(null);

  async function onFetchOrder(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    if (lookupOrderId.trim() === "") {
      setMessage("入力内容を確認してください");
      return;
    }

    const result = await fetchOrder(lookupOrderId.trim());
    if (result.kind === "success") {
      setOrder(result.order);
      setMessage("注文を取得しました");
      return;
    }
    if (result.kind === "notFound") {
      setOrder(null);
      setMessage("注文が見つかりません");
      return;
    }
    setOrder(null);
    setMessage("注文取得に失敗しました");
  }

  async function onCancelOrder(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    if (cancelOrderId.trim() === "") {
      setMessage("入力内容を確認してください");
      return;
    }

    const result = await cancelOrder(cancelOrderId.trim());
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

    const parsedAmount = Number(amount);
    if (
      paymentOrderId.trim() === "" ||
      idempotencyKey.trim() === "" ||
      amount.trim() === "" ||
      !Number.isInteger(parsedAmount) ||
      parsedAmount < 1
    ) {
      setMessage("入力内容を確認してください");
      return;
    }

    const result = await confirmPayment({
      orderId: paymentOrderId.trim(),
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

      <form onSubmit={onFetchOrder} className="stack-form">
        <label htmlFor="lookupOrderId">参照orderId</label>
        <input
          id="lookupOrderId"
          name="lookupOrderId"
          value={lookupOrderId}
          onChange={(event) => setLookupOrderId(event.target.value)}
        />
        <button type="submit">注文を取得</button>
      </form>

      <form onSubmit={onCancelOrder} className="stack-form">
        <label htmlFor="cancelOrderId">キャンセルorderId</label>
        <input
          id="cancelOrderId"
          name="cancelOrderId"
          value={cancelOrderId}
          onChange={(event) => setCancelOrderId(event.target.value)}
        />
        <button type="submit">注文をキャンセル</button>
      </form>

      <form onSubmit={onConfirmPayment} className="stack-form">
        <label htmlFor="paymentOrderId">決済orderId</label>
        <input
          id="paymentOrderId"
          name="paymentOrderId"
          value={paymentOrderId}
          onChange={(event) => setPaymentOrderId(event.target.value)}
        />
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
