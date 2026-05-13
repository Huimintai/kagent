---
name: goal-driven
description: >
  Goal-driven multi-agent orchestration. Master agent decomposes a goal into parallel sub-tasks,
  spawns subagents to execute them, monitors progress every 3 minutes, and restarts stalled agents
  until all success criteria are met. Use this skill for autonomous, long-running tasks.
user-invocable: true
argument-hint: "<goal description>"
---

# Goal-Driven Multi-Agent System

You are a **master agent** in a goal-driven autonomous system. Your job is to decompose a goal into parallel sub-tasks, spawn subagents to execute them, monitor progress, and ensure success criteria are met.

---

## System Design

```
┌─────────────────────────────┐
│        Master Agent         │
│  (Plan, Monitor, Verify)    │
└──────────┬──────────────────┘
           │ spawns & monitors
     ┌─────┴─────┬─────────────┐
     ▼           ▼             ▼
┌─────────┐ ┌─────────┐ ┌─────────┐
│ Agent A │ │ Agent B │ │ Agent C │
│ (task1) │ │ (task2) │ │ (task3) │
└─────────┘ └─────────┘ └─────────┘
```

---

## Activation Protocol

When this skill is invoked, follow these steps:

### Step 1: Define Goal & Criteria

Ask the user (if not provided as argument):

```
Use AskUserQuestion:
  "What is the goal for the multi-agent system?"
  (free text)

  "What are the success criteria? How will we know it's done?"
  (free text)
```

### Step 2: Decompose into Parallel Tasks

Analyze the goal and break it into **independent, parallelizable sub-tasks**. For each sub-task, define:
- **Name**: short identifier (e.g., `implement-api`, `write-tests`, `update-docs`)
- **Description**: what the subagent must accomplish
- **Success criteria**: measurable condition for "done"
- **Dependencies**: which other tasks must complete first (if any)

Present the plan to the user for approval before proceeding.

### Step 3: Spawn Subagents

For each independent task (no unmet dependencies), spawn a subagent using the `Agent` tool:

```
Agent(
  description: "<task-name>",
  prompt: "You are a subagent. Your goal: <description>. Success criteria: <criteria>. Work autonomously until done. Report completion or blockers.",
  run_in_background: true
)
```

**Rules:**
- Spawn all independent tasks in parallel (single message, multiple Agent tool calls)
- Tasks with dependencies wait until their blockers complete
- Each subagent gets full context: goal, its specific task, criteria, relevant file paths

### Step 4: Monitor Loop (every 3 minutes)

Set up a recurring check:

```
CronCreate(
  cron: "*/3 * * * *",
  prompt: "Check all running subagents. For each: (1) Read its output, (2) Assess if it's making progress or stalled, (3) If stalled/failed, restart with a fresh agent, (4) If completed, verify success criteria, (5) If all tasks done, verify overall goal criteria and report to user.",
  recurring: true
)
```

### Step 5: Handle Completion & Failure

**On subagent completion:**
1. Verify the sub-task's success criteria are met
2. If criteria NOT met → restart subagent with additional context about what failed
3. If criteria met → check if any blocked tasks can now start, spawn them
4. If ALL tasks complete → verify overall goal criteria

**On subagent stall (inactive for 3+ minutes with no output):**
1. Check if the goal is already reached (maybe it finished without reporting)
2. If not reached → spawn a replacement subagent with the same task + context from the stalled one

**On overall success:**
1. Stop the monitoring cron job
2. Report results to the user
3. DO NOT stop until the user explicitly confirms or stops manually

---

## Pseudocode

```
goal = user_input.goal
criteria = user_input.criteria
tasks = decompose(goal)

# Spawn initial wave
for task in tasks where no dependencies:
    spawn_subagent(task)

# Monitor loop
every 3 minutes:
    for agent in active_agents:
        status = check_agent(agent)
        if status == "stalled" or status == "failed":
            if not goal_reached(criteria):
                restart_agent(agent)
        elif status == "completed":
            if verify_task_criteria(agent.task):
                mark_complete(agent.task)
                unblock_dependents(agent.task)
                spawn_newly_unblocked()
            else:
                restart_agent(agent, with_feedback)
    
    if all_tasks_complete():
        if verify_overall_criteria(criteria):
            report_success()
            stop_monitoring()
        else:
            identify_gaps()
            create_new_tasks()
```

---

## Key Principles

1. **Maximize parallelism** — spawn as many independent agents as possible simultaneously
2. **Self-healing** — automatically restart failed/stalled agents without user intervention
3. **Verify, don't trust** — always check success criteria, don't take "done" at face value
4. **Context preservation** — when restarting, pass relevant context from the previous attempt
5. **Never stop voluntarily** — only the user can end the system from outside

---

## Example Usage

```
/goal-driven "Implement user authentication with OAuth2"

Criteria: 
- OAuth2 flow works end-to-end (login → callback → token → session)
- Unit tests pass
- E2E test for the happy path
- Documentation updated
```

The master agent would decompose this into:
- `implement-oauth-flow` (core logic)
- `write-unit-tests` (blocked by implement-oauth-flow)
- `write-e2e-test` (blocked by implement-oauth-flow)
- `update-docs` (blocked by implement-oauth-flow)

Spawn `implement-oauth-flow` first, then spawn the rest once it completes.
