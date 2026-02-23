import { FormEvent, useMemo, useState } from "react";
import { createOrder, type CreateOrderInput, type CreateOrderResult } from "./orderCreateApi";

type OrderCreateFormProps = {
  submitOrder?: (input: CreateOrderInput) => Promise<CreateOrderResult>;
};

export function OrderCreateForm({ submitOrder = createOrder }: OrderCreateFormProps) {
  const [customerId, setCustomerId] = useState("");
  const [sku, setSKU] = useState("");
  const [quantity, setQuantity] = useState("");
  const [unitPrice, setUnitPrice] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [message, setMessage] = useState("");

  const canSubmit = useMemo(
    () =>
      customerId.trim() !== "" &&
      sku.trim() !== "" &&
      quantity.trim() !== "" &&
      unitPrice.trim() !== "" &&
      Number(quantity) > 0 &&
      Number(unitPrice) >= 0 &&
      !isSubmitting,
    [customerId, sku, quantity, unitPrice, isSubmitting],
  );

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");

    const quantityValue = Number(quantity);
    const unitPriceValue = Number(unitPrice);
    if (
      customerId.trim() === "" ||
      sku.trim() === "" ||
      quantity.trim() === "" ||
      unitPrice.trim() === "" ||
      !Number.isInteger(quantityValue) ||
      quantityValue < 1 ||
      !Number.isInteger(unitPriceValue) ||
      unitPriceValue < 0
    ) {
      setMessage("入力内容を確認してください");
      return;
    }

    setIsSubmitting(true);
    const result = await submitOrder({
      customerId: customerId.trim(),
      sku: sku.trim(),
      quantity: quantityValue,
      unitPrice: unitPriceValue,
    });
    setIsSubmitting(false);

    if (result.kind === "success") {
      setMessage(`注文を受け付けました: ${result.orderId}`);
      return;
    }
    if (result.kind === "invalid") {
      setMessage("入力内容を確認してください");
      return;
    }
    if (result.kind === "priceConflict") {
      setMessage("価格が更新されました。再確認してください");
      return;
    }
    setMessage("注文作成に失敗しました");
  }

  return (
    <section className="panel panel-create">
      <h2>注文作成</h2>
      <form onSubmit={onSubmit} className="stack-form">
        <div>
          <label htmlFor="customerId">customerId</label>
          <input
            id="customerId"
            name="customerId"
            value={customerId}
            onChange={(event) => setCustomerId(event.target.value)}
          />
        </div>
        <div>
          <label htmlFor="sku">sku</label>
          <input id="sku" name="sku" value={sku} onChange={(event) => setSKU(event.target.value)} />
        </div>
        <div>
          <label htmlFor="quantity">quantity</label>
          <input
            id="quantity"
            name="quantity"
            type="number"
            inputMode="numeric"
            value={quantity}
            onChange={(event) => setQuantity(event.target.value)}
          />
        </div>
        <div>
          <label htmlFor="unitPrice">unitPrice</label>
          <input
            id="unitPrice"
            name="unitPrice"
            type="number"
            inputMode="numeric"
            value={unitPrice}
            onChange={(event) => setUnitPrice(event.target.value)}
          />
        </div>
        <button type="submit" disabled={!canSubmit}>
          注文を作成
        </button>
      </form>
      {message !== "" ? (
        <p role="status" className="status-message">
          {message}
        </p>
      ) : null}
    </section>
  );
}
