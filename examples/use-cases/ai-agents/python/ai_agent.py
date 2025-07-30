"""
AI Agent with Route Prioritization and Tool Orchestration

This example demonstrates:
- Intelligent query routing and prioritization
- Deterministic tool calling with built-in orchestration
- State management and failure handling
- Safety constraints and guardrails
"""

import json
import time
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
from enum import Enum

from hatchet_sdk import Hatchet, Context


class QueryPriority(Enum):
    URGENT = "urgent"
    HIGH = "high" 
    NORMAL = "normal"
    LOW = "low"


@dataclass
class CustomerQuery:
    query_id: str
    customer_id: str
    message: str
    priority: QueryPriority
    context: Dict[str, Any]


@dataclass
class ToolResult:
    tool_name: str
    success: bool
    result: Any
    execution_time: float


@dataclass
class AgentState:
    query_id: str
    current_step: str
    tool_results: List[ToolResult]
    conversation_history: List[Dict[str, str]]
    safety_violations: List[str]


hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["customer.query"])
class AIAgentWorkflow:
    """
    AI Agent workflow with intelligent routing and tool orchestration
    """

    @hatchet.step(timeout="30s", retries=2)
    def route_and_prioritize(self, context: Context) -> dict:
        """Route and prioritize customer queries intelligently"""
        
        data = context.workflow_input()
        query = CustomerQuery(**data)
        
        # Intelligent routing based on query content and customer tier
        route_decision = {
            "query_id": query.query_id,
            "assigned_agent": self._determine_agent_type(query),
            "priority_score": self._calculate_priority_score(query),
            "estimated_complexity": self._assess_complexity(query.message),
            "required_tools": self._identify_required_tools(query.message)
        }
        
        print(f"Query {query.query_id} routed to {route_decision['assigned_agent']} with priority {route_decision['priority_score']}")
        
        return route_decision

    @hatchet.step(timeout="60s", retries=1, parents=["route_and_prioritize"])
    def execute_tools(self, context: Context) -> dict:
        """Execute deterministic tooling with orchestration primitives"""
        
        route_info = context.step_output("route_and_prioritize")
        required_tools = route_info["required_tools"]
        
        state = AgentState(
            query_id=route_info["query_id"],
            current_step="execute_tools",
            tool_results=[],
            conversation_history=[],
            safety_violations=[]
        )
        
        # Execute tools with safety constraints
        for tool_name in required_tools:
            try:
                # Safety check before tool execution
                if not self._check_safety_constraints(tool_name, context):
                    state.safety_violations.append(f"Safety violation prevented {tool_name} execution")
                    continue
                
                start_time = time.time()
                result = self._execute_tool(tool_name, context)
                execution_time = time.time() - start_time
                
                tool_result = ToolResult(
                    tool_name=tool_name,
                    success=True,
                    result=result,
                    execution_time=execution_time
                )
                state.tool_results.append(tool_result)
                
                print(f"Tool {tool_name} executed successfully in {execution_time:.2f}s")
                
            except Exception as e:
                print(f"Tool {tool_name} failed: {str(e)}")
                state.tool_results.append(ToolResult(
                    tool_name=tool_name,
                    success=False,
                    result=str(e),
                    execution_time=0
                ))
        
        return {
            "state": state.__dict__,
            "successful_tools": len([r for r in state.tool_results if r.success]),
            "failed_tools": len([r for r in state.tool_results if not r.success]),
            "safety_violations": len(state.safety_violations)
        }

    @hatchet.step(timeout="45s", parents=["execute_tools"])
    def generate_response(self, context: Context) -> dict:
        """Generate final response with state management"""
        
        tool_results = context.step_output("execute_tools")
        route_info = context.step_output("route_and_prioritize")
        
        # Check if human intervention is needed
        needs_human = (
            tool_results["failed_tools"] > 0 or
            tool_results["safety_violations"] > 0 or
            route_info["priority_score"] > 8
        )
        
        if needs_human:
            # Signal for Human-in-the-Loop
            response = {
                "status": "human_review_required",
                "reason": "Complex query or safety concerns detected",
                "query_id": route_info["query_id"],
                "escalation_priority": "high" if tool_results["safety_violations"] > 0 else "normal"
            }
        else:
            # Generate automated response
            response = {
                "status": "completed",
                "query_id": route_info["query_id"],
                "response": self._generate_final_response(tool_results, route_info),
                "confidence_score": self._calculate_confidence(tool_results),
                "processing_time": time.time()
            }
        
        print(f"Query {route_info['query_id']} processed with status: {response['status']}")
        
        return response

    def _determine_agent_type(self, query: CustomerQuery) -> str:
        """Determine the appropriate agent type based on query characteristics"""
        if "technical" in query.message.lower() or "api" in query.message.lower():
            return "technical_agent"
        elif "billing" in query.message.lower() or "payment" in query.message.lower():
            return "billing_agent"
        else:
            return "general_agent"

    def _calculate_priority_score(self, query: CustomerQuery) -> int:
        """Calculate priority score (1-10 scale)"""
        base_score = {
            QueryPriority.URGENT: 9,
            QueryPriority.HIGH: 7,
            QueryPriority.NORMAL: 5,
            QueryPriority.LOW: 3
        }[query.priority]
        
        # Adjust based on customer tier
        if query.context.get("customer_tier") == "enterprise":
            base_score += 2
        elif query.context.get("customer_tier") == "pro":
            base_score += 1
            
        return min(base_score, 10)

    def _assess_complexity(self, message: str) -> str:
        """Assess query complexity"""
        word_count = len(message.split())
        technical_terms = ["api", "integration", "webhook", "database", "authentication"]
        
        if word_count > 100 or any(term in message.lower() for term in technical_terms):
            return "high"
        elif word_count > 50:
            return "medium"
        else:
            return "low"

    def _identify_required_tools(self, message: str) -> List[str]:
        """Identify required tools based on message content"""
        tools = []
        
        if "status" in message.lower() or "health" in message.lower():
            tools.append("system_status_checker")
            
        if "account" in message.lower() or "billing" in message.lower():
            tools.append("account_lookup")
            
        if "documentation" in message.lower() or "how to" in message.lower():
            tools.append("knowledge_base_search")
            
        if "api" in message.lower():
            tools.append("api_validator")
            
        return tools if tools else ["general_assistant"]

    def _check_safety_constraints(self, tool_name: str, context: Context) -> bool:
        """Check safety constraints before tool execution"""
        
        # Example safety constraints
        dangerous_tools = ["system_admin", "database_modifier"]
        
        if tool_name in dangerous_tools:
            # Check if user has appropriate permissions
            user_role = context.workflow_input().get("context", {}).get("user_role", "user")
            if user_role not in ["admin", "superuser"]:
                return False
                
        # Rate limiting check
        if self._check_rate_limit(tool_name, context):
            return False
            
        return True

    def _check_rate_limit(self, tool_name: str, context: Context) -> bool:
        """Check if tool execution would exceed rate limits"""
        # Simplified rate limiting logic
        # In practice, this would check Redis or similar
        return False

    def _execute_tool(self, tool_name: str, context: Context) -> Any:
        """Execute the specified tool"""
        
        # Mock tool implementations
        tools = {
            "system_status_checker": lambda: {"status": "operational", "uptime": "99.9%"},
            "account_lookup": lambda: {"account_id": "12345", "tier": "pro", "status": "active"},
            "knowledge_base_search": lambda: {"results": ["Doc 1", "Doc 2"], "confidence": 0.85},
            "api_validator": lambda: {"valid": True, "last_tested": "2024-01-01"},
            "general_assistant": lambda: {"response": "I can help you with that!"}
        }
        
        if tool_name in tools:
            return tools[tool_name]()
        else:
            raise Exception(f"Unknown tool: {tool_name}")

    def _generate_final_response(self, tool_results: dict, route_info: dict) -> str:
        """Generate the final response based on tool results"""
        
        successful_tools = tool_results["successful_tools"]
        
        if successful_tools == 0:
            return "I apologize, but I encountered issues processing your request. Please try again or contact support."
        
        return f"Based on my analysis using {successful_tools} tools, I can help you with your query. Here's what I found..."

    def _calculate_confidence(self, tool_results: dict) -> float:
        """Calculate confidence score for the response"""
        
        total_tools = tool_results["successful_tools"] + tool_results["failed_tools"]
        if total_tools == 0:
            return 0.0
            
        success_rate = tool_results["successful_tools"] / total_tools
        
        # Penalize for safety violations
        if tool_results["safety_violations"] > 0:
            success_rate *= 0.7
            
        return round(success_rate, 2)


if __name__ == "__main__":
    # Example usage
    worker = hatchet.worker("ai-agent-worker", workflows=[AIAgentWorkflow()])
    worker.start()