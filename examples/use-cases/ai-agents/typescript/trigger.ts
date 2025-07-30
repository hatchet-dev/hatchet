/**
 * Trigger examples for AI Agent workflow
 */

import { Hatchet } from '@hatchet-dev/typescript-sdk/v1';

const hatchet = new Hatchet();

async function main() {
  // Example 1: Technical integration query
  await hatchet.admin.runWorkflow('ai-agent-workflow', {
    queryId: 'tech-query-001',
    customerId: 'enterprise-customer-123',
    message: 'Our webhook endpoints are receiving malformed payloads. Can you help diagnose the API integration issues?',
    priority: 'high',
    context: {
      customerTier: 'enterprise',
      userRole: 'developer',
      integrationType: 'webhook'
    }
  });

  // Example 2: Billing inquiry
  await hatchet.admin.runWorkflow('ai-agent-workflow', {
    queryId: 'billing-query-002', 
    customerId: 'pro-customer-456',
    message: 'I need to understand the charges on my latest invoice. There are some API call overages I want to review.',
    priority: 'normal',
    context: {
      customerTier: 'pro',
      userRole: 'admin'
    }
  });

  // Example 3: Complex research requiring human review
  await hatchet.admin.runWorkflow('ai-agent-workflow', {
    queryId: 'research-query-003',
    customerId: 'startup-customer-789', 
    message: 'I need comprehensive guidance on implementing a distributed task queue system with Hatchet for autonomous AI agents, including safety constraints, tool orchestration, and human-in-the-loop workflows for a multi-tenant environment.',
    priority: 'urgent',
    context: {
      customerTier: 'starter',
      userRole: 'cto',
      projectScale: 'enterprise'
    }
  });

  // Example 4: System status inquiry
  await hatchet.admin.runWorkflow('ai-agent-workflow', {
    queryId: 'status-query-004',
    customerId: 'pro-customer-101',
    message: 'Are there any ongoing issues with the API? My workflows have been failing intermittently.',
    priority: 'high',
    context: {
      customerTier: 'pro',
      userRole: 'engineer'
    }
  });

  console.log('Triggered AI agent workflows');
}

if (require.main === module) {
  main().catch(console.error);
}