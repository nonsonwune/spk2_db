# Multi-agent system implementing a hierarchical AI team structure
# Each agent has a specialized role in problem-solving

import ollama
from typing import Dict
import time
import asyncio
from dataclasses import dataclass

@dataclass
class AgentResponse:
    """Simple response structure"""
    content: str
    agent_name: str

class AIAgent:
    """Simplified AI agent class"""
    
    def __init__(self, name: str, system_prompt: str, model: str = "llama3.2:3b"):
        self.name = name
        self.system_prompt = system_prompt
        self.model = model
    
    async def think(self, input_text: str) -> AgentResponse:
        """Process the input and return a response"""
        try:
            response = ollama.chat(
                model=self.model,
                messages=[
                    {"role": "system", "content": self.system_prompt},
                    {"role": "user", "content": input_text}
                ]
            )
            return AgentResponse(
                content=response["message"]["content"],
                agent_name=self.name
            )
        except Exception as e:
            print(f"Error in {self.name}: {str(e)}")
            return AgentResponse(
                content=f"Error: {str(e)}",
                agent_name=self.name
            )

class AITeam:
    """Simplified team orchestrator"""
    
    def __init__(self):
        self.agents: Dict[str, AIAgent] = {}
        self._create_agents()
    
    def _create_agents(self):
        """Initialize all agents with their specific roles"""
        
        # CEO Agent
        self.agents["CEO"] = AIAgent(
            "CEO",
            """You are the Chief Executive AI. Break down problems into 4 clear steps.
            When reviewing, provide a concise summary of all implementations."""
        )
        
        # Specialist agents
        specialists = [
            ("O1", "You are the Foundation Specialist. Focus ONLY on implementing step 1 of the plan."),
            ("O2", "You are the Builder Specialist. Focus ONLY on implementing step 2 of the plan."),
            ("O3", "You are the Integration Specialist. Focus ONLY on implementing step 3 of the plan."),
            ("O4", "You are the Quality Specialist. Focus ONLY on implementing step 4 of the plan.")
        ]
        
        for name, prompt in specialists:
            self.agents[name] = AIAgent(name, prompt)
    
    async def process_request(self, user_input: str) -> Dict[str, str]:
        """Process a user request through the AI team"""
        
        print("\n🤖 Processing request...\n")
        
        # Get initial plan from CEO
        initial_plan = await self.agents["CEO"].think(user_input)
        print("📋 Initial Plan:")
        print("-" * 50)
        print(initial_plan.content)
        print("-" * 50)
        
        # Get specialist implementations
        specialist_responses = []
        for i in range(1, 5):
            response = await self.agents[f"O{i}"].think(initial_plan.content)
            specialist_responses.append(response)
            print(f"\n👉 Step {i} Implementation:")
            print("-" * 50)
            print(response.content)
            print("-" * 50)
        
        # Prepare final review
        final_review = "\n".join([
            "=== Original Plan ===",
            initial_plan.content,
            "=== Implementations ===",
            *[f"Step {i+1}: {resp.content}" for i, resp in enumerate(specialist_responses)]
        ])
        
        # Get final summary
        final_summary = await self.agents["CEO"].think(final_review)
        print("\n📊 Executive Summary:")
        print("-" * 50)
        print(final_summary.content)
        print("-" * 50)
        
        return {
            "initial_plan": initial_plan.content,
            "step_1": specialist_responses[0].content,
            "step_2": specialist_responses[1].content,
            "step_3": specialist_responses[2].content,
            "step_4": specialist_responses[3].content,
            "final_summary": final_summary.content
        }

async def main():
    team = AITeam()
    problem = "write a python code to refresh https://naija-find-uni.onrender.com/ every 5 seconds"
    
    print("\n🎯 Task:", problem)
    await team.process_request(problem)

if __name__ == "__main__":
    asyncio.run(main())