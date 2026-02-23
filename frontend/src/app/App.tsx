import { OrderCreateForm } from "../features/order-create/OrderCreateForm";
import { OrderActionsPanel } from "../features/order-actions/OrderActionsPanel";
import { MockLanePanels } from "../features/mock-lane/MockLanePanels";

export function App() {
  return (
    <main>
      <h1>注文作成</h1>
      <OrderCreateForm />
      <OrderActionsPanel />
      <MockLanePanels />
    </main>
  );
}
