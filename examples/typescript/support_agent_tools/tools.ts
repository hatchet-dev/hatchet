// > Setup
import { z } from 'zod/v4';
import { hatchet } from '../hatchet-client';

// > Models
const CustomerLookupInput = z.object({
  customerId: z.string(),
});

type CustomerLookupInputType = z.infer<typeof CustomerLookupInput>;

type CustomerInfo = {
  customerId: string;
  name: string;
  email: string;
  plan: string;
  accountStatus: string;
  defaultOrderId: string;
  supportTier: string;
};

const OrderStatusInput = z.object({
  orderId: z.string(),
});

type OrderStatusInputType = z.infer<typeof OrderStatusInput>;

type OrderStatus = {
  orderId: string;
  status: string;
  lastUpdated: string;
  estimatedDelivery: string;
  knownIssue: string | null;
  carrier: string;
  trackingNumber: string;
};

const CreateTicketInput = z.object({
  customerId: z.string(),
  orderId: z.string(),
  subject: z.string(),
  body: z.string(),
  priority: z.string(),
});

type CreateTicketInputType = z.infer<typeof CreateTicketInput>;

type TicketResult = {
  ticketId: string;
  status: string;
  priority: string;
  routingTeam: string;
  summary: string;
};

// > Fixture data
const CUSTOMERS: Record<string, CustomerInfo> = {
  'C-100': {
    customerId: 'C-100',
    name: 'Alice Martin',
    email: 'alice@example.com',
    plan: 'business',
    accountStatus: 'active',
    defaultOrderId: 'ORD-9987',
    supportTier: 'priority',
  },
};

const ORDERS: Record<string, OrderStatus> = {
  'ORD-9987': {
    orderId: 'ORD-9987',
    status: 'delayed',
    lastUpdated: '2026-05-20T14:30:00Z',
    estimatedDelivery: '2026-05-28',
    knownIssue: 'Carrier reported weather delay at regional hub',
    carrier: 'FastShip',
    trackingNumber: 'FS-482910',
  },
};

// > Lookup customer
export const lookupCustomer = hatchet.task({
  name: 'lookup-customer',
  inputValidator: CustomerLookupInput,
  description: 'Look up a customer by ID and return their profile, plan, and support tier.',
  fn: async (input: CustomerLookupInputType): Promise<CustomerInfo> => {
    const customer = CUSTOMERS[input.customerId];
    if (!customer) {
      return {
        customerId: input.customerId,
        name: 'Unknown',
        email: 'unknown@example.com',
        plan: 'none',
        accountStatus: 'not_found',
        defaultOrderId: '',
        supportTier: 'standard',
      };
    }
    return customer;
  },
});

// > Check order status
export const checkOrderStatus = hatchet.task({
  name: 'check-order-status',
  inputValidator: OrderStatusInput,
  description: 'Check the current status, carrier, and any known issues for an order.',
  fn: async (input: OrderStatusInputType): Promise<OrderStatus> => {
    const order = ORDERS[input.orderId];
    if (!order) {
      return {
        orderId: input.orderId,
        status: 'not_found',
        lastUpdated: '',
        estimatedDelivery: '',
        knownIssue: null,
        carrier: 'unknown',
        trackingNumber: '',
      };
    }
    return order;
  },
});

// > Create ticket
export const createTicket = hatchet.task({
  name: 'create-ticket',
  inputValidator: CreateTicketInput,
  description: 'Create a support ticket for a customer issue and return the ticket ID and routing.',
  fn: async (input: CreateTicketInputType): Promise<TicketResult> => {
    const ticketId = `TICKET-${input.customerId}-001`;
    return {
      ticketId,
      status: 'open',
      priority: input.priority,
      routingTeam: 'shipping-support',
      summary: `Ticket ${ticketId} created for ${input.customerId} regarding order ${input.orderId}: ${input.subject}`,
    };
  },
});

// > Create Claude tools
export function createLookupCustomerToolClaude() {
  return lookupCustomer.mcpTool('claude');
}

export function createCheckOrderStatusToolClaude() {
  return checkOrderStatus.mcpTool('claude');
}

export function createTicketToolClaude() {
  return createTicket.mcpTool('claude');
}

// > Create openai tools
export function createLookupCustomerToolOpenai() {
  return lookupCustomer.mcpTool('openai');
}

export function createCheckOrderStatusToolOpenai() {
  return checkOrderStatus.mcpTool('openai');
}

export function createTicketToolOpenai() {
  return createTicket.mcpTool('openai');
}
