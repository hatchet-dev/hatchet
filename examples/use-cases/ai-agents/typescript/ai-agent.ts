/**
 * AI Agent with Streaming and Tool Orchestration
 * 
 * This example demonstrates:
 * - Complex tool call orchestration with timeouts
 * - Conversation state management
 * - Built-in streaming capabilities 
 * - Human-in-the-Loop signaling
 */

import { Hatchet, Context } from '@hatchet-dev/typescript-sdk/v1';

interface CustomerQuery {
  queryId: string;
  customerId: string;
  message: string;
  priority: 'urgent' | 'high' | 'normal' | 'low';
  context: Record<string, any>;
}

interface ToolCall {
  name: string;
  parameters: Record<string, any>;
  timeout: number;
}

interface ConversationState {
  queryId: string;
  messages: Array<{ role: string; content: string; timestamp: number }>;
  toolResults: Array<{ tool: string; success: boolean; result: any; duration: number }>;
  safetyFlags: string[];
  humanReviewRequired: boolean;
}

const hatchet = new Hatchet();

export default hatchet.workflow('ai-agent-workflow', async (context: Context) => {
  return {
    // Step 1: Analyze and route the query
    analyzeQuery: hatchet.step('analyze-query', {
      timeout: '30s',
      retries: 2
    })(async (context: Context): Promise<{
      routing: any;
      toolPlan: ToolCall[];
      riskAssessment: any;
    }> => {
      const query = context.workflowInput() as CustomerQuery;
      
      console.log(`Analyzing query ${query.queryId} from customer ${query.customerId}`);
      
      // Intelligent query analysis
      const routing = analyzeQueryIntent(query);
      const toolPlan = planToolExecution(query);
      const riskAssessment = assessSecurityRisk(query);
      
      return {
        routing,
        toolPlan, 
        riskAssessment
      };
    }),

    // Step 2: Execute tools with orchestration
    executeTools: hatchet.step('execute-tools', {
      timeout: '120s',
      retries: 1,
      parents: ['analyze-query']
    })(async (context: Context): Promise<ConversationState> => {
      const { toolPlan, riskAssessment } = context.stepOutput('analyze-query');
      const query = context.workflowInput() as CustomerQuery;
      
      const state: ConversationState = {
        queryId: query.queryId,
        messages: [{
          role: 'user',
          content: query.message,
          timestamp: Date.now()
        }],
        toolResults: [],
        safetyFlags: [],
        humanReviewRequired: false
      };

      // Execute tools with safety constraints
      for (const toolCall of toolPlan) {
        try {
          // Safety check
          if (!checkToolSafety(toolCall, riskAssessment)) {
            state.safetyFlags.push(`Blocked ${toolCall.name} due to safety constraints`);
            continue;
          }

          const startTime = Date.now();
          
          // Execute with timeout
          const result = await executeToolWithTimeout(toolCall, context);
          const duration = Date.now() - startTime;
          
          state.toolResults.push({
            tool: toolCall.name,
            success: true,
            result,
            duration
          });

          console.log(`Tool ${toolCall.name} completed in ${duration}ms`);
          
        } catch (error) {
          state.toolResults.push({
            tool: toolCall.name,
            success: false,
            result: error.message,
            duration: 0
          });
          
          console.error(`Tool ${toolCall.name} failed:`, error.message);
        }
      }

      // Determine if human review is needed
      state.humanReviewRequired = shouldRequireHumanReview(state, riskAssessment);

      return state;
    }),

    // Step 3: Generate response with streaming
    generateResponse: hatchet.step('generate-response', {
      timeout: '60s',
      parents: ['execute-tools']
    })(async (context: Context): Promise<{
      response: any;
      streamingEnabled: boolean;
      humanEscalation?: any;
    }> => {
      const state = context.stepOutput('execute-tools') as ConversationState;
      const { routing } = context.stepOutput('analyze-query');
      
      if (state.humanReviewRequired) {
        // Signal for Human-in-the-Loop
        return {
          response: {
            status: 'escalated',
            queryId: state.queryId,
            reason: 'Complex query requires human expertise',
            summary: generateExecutionSummary(state)
          },
          streamingEnabled: false,
          humanEscalation: {
            priority: routing.priority,
            context: state,
            suggestedActions: generateHumanActions(state)
          }
        };
      }

      // Generate streaming response
      const response = await generateStreamingResponse(state, context);
      
      return {
        response,
        streamingEnabled: true
      };
    })
  };
});

// Helper functions
function analyzeQueryIntent(query: CustomerQuery) {
  const intent = {
    category: categorizeQuery(query.message),
    complexity: assessComplexity(query.message),
    urgency: query.priority,
    customerTier: query.context.customerTier || 'standard'
  };

  return {
    ...intent,
    routingDecision: determineRouting(intent),
    estimatedTime: estimateProcessingTime(intent)
  };
}

function planToolExecution(query: CustomerQuery): ToolCall[] {
  const tools: ToolCall[] = [];
  const message = query.message.toLowerCase();

  // Knowledge base search for documentation queries
  if (message.includes('how to') || message.includes('documentation')) {
    tools.push({
      name: 'knowledge-search',
      parameters: { query: query.message, limit: 5 },
      timeout: 10000
    });
  }

  // System status for health/status queries
  if (message.includes('status') || message.includes('down') || message.includes('error')) {
    tools.push({
      name: 'system-health-check',
      parameters: { services: ['api', 'database', 'cache'] },
      timeout: 15000
    });
  }

  // Account lookup for billing/account queries
  if (message.includes('account') || message.includes('billing')) {
    tools.push({
      name: 'account-lookup',
      parameters: { customerId: query.customerId },
      timeout: 5000
    });
  }

  // Code analysis for technical queries
  if (message.includes('api') || message.includes('integration')) {
    tools.push({
      name: 'code-analyzer',
      parameters: { context: query.context },
      timeout: 20000
    });
  }

  return tools.length > 0 ? tools : [{
    name: 'general-assistant',
    parameters: { query: query.message },
    timeout: 10000
  }];
}

function assessSecurityRisk(query: CustomerQuery) {
  const riskFactors = [];
  const message = query.message.toLowerCase();

  // Check for sensitive operations
  if (message.includes('delete') || message.includes('remove')) {
    riskFactors.push('destructive_operation');
  }

  if (message.includes('admin') || message.includes('root')) {
    riskFactors.push('elevated_privileges');
  }

  // Customer tier risk assessment
  const tierRisk = {
    'enterprise': 'low',
    'pro': 'medium', 
    'starter': 'high'
  }[query.context.customerTier] || 'high';

  return {
    level: riskFactors.length > 0 ? 'high' : tierRisk,
    factors: riskFactors,
    requiresApproval: riskFactors.length > 1
  };
}

function checkToolSafety(toolCall: ToolCall, riskAssessment: any): boolean {
  // High-risk tools require additional checks
  const highRiskTools = ['system-admin', 'database-query', 'user-management'];
  
  if (highRiskTools.includes(toolCall.name) && riskAssessment.level === 'high') {
    return false;
  }

  return true;
}

async function executeToolWithTimeout(toolCall: ToolCall, context: Context): Promise<any> {
  const tools = {
    'knowledge-search': async (params: any) => ({
      results: [
        { title: 'API Documentation', relevance: 0.95 },
        { title: 'Integration Guide', relevance: 0.87 }
      ],
      totalFound: 2
    }),
    
    'system-health-check': async (params: any) => ({
      status: 'operational',
      services: {
        api: { status: 'healthy', responseTime: '45ms' },
        database: { status: 'healthy', connections: 12 },
        cache: { status: 'healthy', hitRate: '94%' }
      }
    }),
    
    'account-lookup': async (params: any) => ({
      customerId: params.customerId,
      tier: 'pro',
      status: 'active',
      usage: { current: '1,250', limit: '10,000' }
    }),
    
    'code-analyzer': async (params: any) => ({
      analysis: 'Code structure looks good',
      suggestions: ['Add error handling', 'Implement rate limiting'],
      score: 85
    }),
    
    'general-assistant': async (params: any) => ({
      response: 'I can help you with that. Let me analyze your request...',
      confidence: 0.9
    })
  };

  const executor = tools[toolCall.name];
  if (!executor) {
    throw new Error(`Unknown tool: ${toolCall.name}`);
  }

  return new Promise(async (resolve, reject) => {
    const timer = setTimeout(() => {
      reject(new Error(`Tool ${toolCall.name} timed out after ${toolCall.timeout}ms`));
    }, toolCall.timeout);

    try {
      const result = await executor(toolCall.parameters);
      clearTimeout(timer);
      resolve(result);
    } catch (error) {
      clearTimeout(timer);
      reject(error);
    }
  });
}

function shouldRequireHumanReview(state: ConversationState, riskAssessment: any): boolean {
  // Require human review if:
  // - High risk assessment
  // - Multiple tool failures
  // - Safety flags raised
  // - Complex query with low confidence
  
  const failedTools = state.toolResults.filter(r => !r.success).length;
  
  return (
    riskAssessment.level === 'high' ||
    failedTools > 1 ||
    state.safetyFlags.length > 0 ||
    riskAssessment.requiresApproval
  );
}

async function generateStreamingResponse(state: ConversationState, context: Context): Promise<any> {
  const successfulResults = state.toolResults.filter(r => r.success);
  
  // Simulate streaming response generation
  const responseChunks = [
    'Based on my analysis',
    'I found the following information',
    'Here are the key points',
    'Let me know if you need clarification'
  ];

  return {
    status: 'completed',
    queryId: state.queryId,
    message: responseChunks.join(' '),
    toolsUsed: successfulResults.length,
    confidence: calculateConfidence(state),
    processingTime: state.toolResults.reduce((sum, r) => sum + r.duration, 0),
    streamingChunks: responseChunks
  };
}

function generateExecutionSummary(state: ConversationState): string {
  const successful = state.toolResults.filter(r => r.success).length;
  const failed = state.toolResults.filter(r => !r.success).length;
  
  return `Executed ${successful + failed} tools (${successful} successful, ${failed} failed). ${state.safetyFlags.length} safety flags raised.`;
}

function generateHumanActions(state: ConversationState): string[] {
  const actions = [];
  
  if (state.toolResults.some(r => !r.success)) {
    actions.push('Review failed tool executions');
  }
  
  if (state.safetyFlags.length > 0) {
    actions.push('Assess safety concerns');
  }
  
  actions.push('Provide expert guidance');
  
  return actions;
}

function categorizeQuery(message: string): string {
  const categories = {
    'technical': ['api', 'integration', 'webhook', 'error', 'bug'],
    'billing': ['billing', 'payment', 'invoice', 'subscription'],
    'general': ['help', 'question', 'how to']
  };

  for (const [category, keywords] of Object.entries(categories)) {
    if (keywords.some(keyword => message.toLowerCase().includes(keyword))) {
      return category;
    }
  }

  return 'general';
}

function assessComplexity(message: string): 'low' | 'medium' | 'high' {
  const wordCount = message.split(' ').length;
  const technicalTerms = ['api', 'webhook', 'authentication', 'database', 'integration'];
  const hasTechnicalTerms = technicalTerms.some(term => message.toLowerCase().includes(term));

  if (wordCount > 100 || hasTechnicalTerms) return 'high';
  if (wordCount > 50) return 'medium';
  return 'low';
}

function determineRouting(intent: any): string {
  if (intent.category === 'technical' && intent.complexity === 'high') {
    return 'senior-technical-agent';
  }
  if (intent.category === 'billing') {
    return 'billing-specialist';
  }
  return 'general-agent';
}

function estimateProcessingTime(intent: any): number {
  const baseTime = 30; // seconds
  const complexityMultiplier = {
    'low': 1,
    'medium': 1.5,
    'high': 2.5
  }[intent.complexity];

  return Math.round(baseTime * complexityMultiplier);
}

function calculateConfidence(state: ConversationState): number {
  const totalTools = state.toolResults.length;
  if (totalTools === 0) return 0;

  const successRate = state.toolResults.filter(r => r.success).length / totalTools;
  
  // Penalize for safety flags
  const safetyPenalty = state.safetyFlags.length * 0.1;
  
  return Math.max(0, Math.round((successRate - safetyPenalty) * 100) / 100);
}