import { OrderCreateForm } from "../features/order-create/OrderCreateForm";
import { OrderActionsPanel } from "../features/order-actions/OrderActionsPanel";

export function App() {
  return (
    <main>
      <h1>注文作成</h1>
      <OrderCreateForm />
      <OrderActionsPanel />
    </main>
  );
}
