"""
Trigger script for AI Agent workflow
"""

import asyncio
from ai_agent import hatchet, QueryPriority

async def main():
    # Example: Trigger different types of customer queries
    
    # High-priority technical query
    await hatchet.admin.run_workflow(
        "AIAgentWorkflow",
        {
            "query_id": "tech-001",
            "customer_id": "cust-enterprise-123",
            "message": "Our API integration is returning 500 errors. Can you check the system status?",
            "priority": QueryPriority.HIGH.value,
            "context": {
                "customer_tier": "enterprise",
                "user_role": "admin"
            }
        }
    )
    
    # Normal billing query
    await hatchet.admin.run_workflow(
        "AIAgentWorkflow", 
        {
            "query_id": "billing-002",
            "customer_id": "cust-pro-456",
            "message": "I need help understanding my billing statement for this month",
            "priority": QueryPriority.NORMAL.value,
            "context": {
                "customer_tier": "pro",
                "user_role": "user"
            }
        }
    )
    
    # Complex research query requiring human review
    await hatchet.admin.run_workflow(
        "AIAgentWorkflow",
        {
            "query_id": "research-003", 
            "customer_id": "cust-startup-789",
            "message": "I need comprehensive documentation on implementing webhooks with custom authentication, rate limiting, and error handling for a multi-tenant SaaS application",
            "priority": QueryPriority.URGENT.value,
            "context": {
                "customer_tier": "startup",
                "user_role": "developer"
            }
        }
    )

if __name__ == "__main__":
    asyncio.run(main())