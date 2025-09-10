# ESXi Commander Agent Configuration

## Overview

Optimized agent configuration based on Anthropic best practices and Ubuntu-only project scope. Replaces the original 3 agents with focused, specialized agents that follow the principle of single responsibility.

## Agent Structure

### 1. esxi-ubuntu-specialist
**Role**: Ubuntu VM lifecycle expert
**Tools**: Read, Grep, Task (minimal read-only + delegation)
**Focus**: 
- 8-command CLI operations only
- Cloud-init guestinfo customization
- govmomi (primary) + SSH (fallback) command generation
- Template validation
- Ubuntu LTS patterns only

**Key Changes from Original**:
- Removed all Windows/Cloudbase-Init sections
- Simplified command surface to match project scope
- Deep focus on cloud-init guestinfo patterns
- Added template validation as first-class concern

### 2. safety-reviewer  
**Role**: Security and production readiness reviewer
**Tools**: Read, Glob, Grep (read-only for safety)
**Focus**:
- AI agent sandboxing validation
- Command injection prevention
- Secret management review
- Backup integrity verification
- Testing coverage analysis

**Key Changes from Original**:
- Renamed from architecture-reviewer
- Added AI agent sandboxing as critical review area
- Focused on Ubuntu-only security concerns
- Structured priority matrix (Critical/High/Medium/Low)

### 3. implementation-assistant
**Role**: Active development and testing
**Tools**: Read, Write, Edit, Bash, Task (full development access)
**Focus**:
- Test-driven development patterns
- Cloud-init YAML generation
- Code implementation with Go patterns
- Integration test writing
- Chaos test scenarios

**Key Changes from Original**:
- Replaced claude-config-maintainer
- Active code generation vs passive documentation
- TDD methodology emphasis
- Ubuntu-specific patterns and examples

## Design Principles Applied

### Anthropic Best Practices
1. **Simplicity**: Each agent has single, clear responsibility
2. **Minimal Tools**: Only necessary permissions per agent
3. **Clear Boundaries**: No overlapping responsibilities
4. **Transparency**: Explicit planning steps in agent prompts

### Project-Specific Optimization
1. **Ubuntu-Only**: No multi-OS branching or Windows support
2. **Cloud-Init Centerpiece**: Deep expertise in guestinfo patterns
3. **8-Command CLI**: Aligned with simplified interface
4. **Safety-First**: Security review prioritized

## Tool Assignment Strategy

- **esxi-ubuntu-specialist**: Read, Grep, Task - Can research and delegate but not modify
- **safety-reviewer**: Read, Glob, Grep - Read-only for unbiased security review
- **implementation-assistant**: Full tools - Active development with all capabilities

## Usage Patterns

### For ESXi Operations
1. Use `esxi-ubuntu-specialist` for command planning and ESXi expertise
2. Use `safety-reviewer` to validate security and safety
3. Use `implementation-assistant` for actual code implementation

### For Development
1. Start with TDD approach via `implementation-assistant`
2. Review code with `safety-reviewer`
3. Validate ESXi command patterns with `esxi-ubuntu-specialist`

## Agent Collaboration

The agents are designed to work together:
- **Specialist** provides ESXi expertise and command plans
- **Reviewer** validates security and production readiness  
- **Assistant** implements code following TDD practices

This creates a balanced team with clear separation of concerns while maintaining comprehensive coverage of the project needs.

## Backup

Original agent configurations are preserved in `.claude/agents.backup/` for reference or rollback if needed.