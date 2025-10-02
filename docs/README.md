# Simple Content Documentation

Welcome to the simple-content project documentation. This directory contains comprehensive guides for developers, operators, and contributors.

## Quick Navigation

### For New Developers
1. Start with [../README.md](../README.md) - Project overview
2. Read [../CLAUDE.md](../CLAUDE.md) - Development conventions
3. Review [Status Lifecycle Guide](#status-lifecycle-documentation) - Core concept

### For Operators
1. [Status Lifecycle Operational Guide](STATUS_LIFECYCLE.md) - Day-to-day operations
2. [Admin Tools Guide](ADMIN_TOOLS.md) - Administrative utilities
3. [Scanner Documentation](SCANNER.md) - Content scanning tools

### For Contributors
1. [Status Lifecycle Refactoring Plan](STATUS_LIFECYCLE_REFACTORING.md) - Current technical debt
2. [Implementation TODO](STATUS_LIFECYCLE_TODO.md) - Sprint tasks

---

## Status Lifecycle Documentation

The status lifecycle is a core concept in simple-content. These three documents work together:

### 1. [STATUS_LIFECYCLE.md](STATUS_LIFECYCLE.md) - Operational Guide ⭐
**Audience:** All developers and operators
**Purpose:** Comprehensive guide to how the status system **should** work

**Contents:**
- Three-tier status model (Content, Object, Derived Content)
- Status state machines and transitions
- Complete lifecycle flows with diagrams
- Database schema and queries
- Best practices and troubleshooting
- Monitoring and debugging queries

**When to use:** Reference this for day-to-day development and operations.

---

### 2. [STATUS_LIFECYCLE_REFACTORING.md](STATUS_LIFECYCLE_REFACTORING.md) - Gap Analysis 🔍
**Audience:** Technical leads, architects, senior developers
**Purpose:** Documents gaps between intended design and current implementation

**Contents:**
- 8 identified implementation gaps with severity ratings
- Detailed gap analysis with code evidence
- Refactoring plan across 3 phases (Foundation, Business Logic, Advanced)
- Code examples for each improvement
- Testing strategy
- Migration path and backward compatibility

**When to use:** Planning sprints, technical debt discussions, architecture reviews.

---

### 3. [STATUS_LIFECYCLE_TODO.md](STATUS_LIFECYCLE_TODO.md) - Implementation Checklist ✅
**Audience:** Developers implementing the refactoring
**Purpose:** Sprint-by-sprint task breakdown

**Contents:**
- Pre-refactoring setup tasks
- Phase 1: Foundation (2 sprints, 10 days)
- Phase 2: Business Logic (2 sprints, 10 days)
- Phase 3: Advanced Features (2-3 sprints, 15 days)
- Detailed checklists for code, tests, docs
- Estimated timeline: ~35 working days (7 weeks)

**When to use:** Daily sprint work, task tracking, implementation reference.

---

## Administrative Tools Documentation

### [ADMIN_TOOLS.md](ADMIN_TOOLS.md)
Guide to administrative CLI tools for database operations:
- Listing content with advanced filters
- Counting and statistics
- Batch operations
- Usage in other projects

### [USING_ADMIN_IN_YOUR_PROJECT.md](USING_ADMIN_IN_YOUR_PROJECT.md)
Step-by-step guide to integrating admin tools into your own Go applications.

### [SCANNER.md](SCANNER.md)
Documentation for the content scanning and processing framework:
- Scanner architecture
- Processor patterns
- Batch operations
- Examples and use cases

---

## Documentation Hierarchy

```
simple-content/
├── README.md                          # Project overview
├── CLAUDE.md                          # AI coding guidelines
├── PROGRAMMATIC_USAGE.md              # Library usage guide
├── Design.md                          # Original design document
└── docs/
    ├── README.md                      # This file
    │
    ├── STATUS_LIFECYCLE.md            # ⭐ Status system guide
    ├── STATUS_LIFECYCLE_REFACTORING.md # 🔍 Gap analysis
    ├── STATUS_LIFECYCLE_TODO.md       # ✅ Implementation tasks
    │
    ├── ADMIN_TOOLS.md                 # Admin CLI tools
    ├── USING_ADMIN_IN_YOUR_PROJECT.md # Admin integration
    └── SCANNER.md                     # Scanner framework
```

---

## How to Read the Documentation

### Scenario: I'm new to the project
1. Read [../README.md](../README.md) for project overview
2. Read [../CLAUDE.md](../CLAUDE.md) for coding conventions
3. Read [STATUS_LIFECYCLE.md](STATUS_LIFECYCLE.md) sections 1-3 for core concepts
4. Skim [ADMIN_TOOLS.md](ADMIN_TOOLS.md) to understand available tools

### Scenario: I need to implement a new feature
1. Check [STATUS_LIFECYCLE.md](STATUS_LIFECYCLE.md) for status handling patterns
2. Review [../CLAUDE.md](../CLAUDE.md) for API design principles
3. Look at examples in [SCANNER.md](SCANNER.md) if using batch operations
4. Check [STATUS_LIFECYCLE_REFACTORING.md](STATUS_LIFECYCLE_REFACTORING.md) for known limitations

### Scenario: I'm debugging a status issue
1. Check [STATUS_LIFECYCLE.md § Troubleshooting](STATUS_LIFECYCLE.md#troubleshooting)
2. Run monitoring queries from [STATUS_LIFECYCLE.md § Monitoring Queries](STATUS_LIFECYCLE.md#monitoring-queries)
3. Check [STATUS_LIFECYCLE_REFACTORING.md](STATUS_LIFECYCLE_REFACTORING.md) for known gaps

### Scenario: I'm planning the status refactoring sprint
1. Review [STATUS_LIFECYCLE_REFACTORING.md](STATUS_LIFECYCLE_REFACTORING.md) for complete context
2. Use [STATUS_LIFECYCLE_TODO.md](STATUS_LIFECYCLE_TODO.md) for sprint planning
3. Reference [STATUS_LIFECYCLE.md](STATUS_LIFECYCLE.md) for target state

### Scenario: I need to query/manage content
1. Check [ADMIN_TOOLS.md](ADMIN_TOOLS.md) for CLI commands
2. Use [STATUS_LIFECYCLE.md § Monitoring Queries](STATUS_LIFECYCLE.md#monitoring-queries) for SQL
3. Review [SCANNER.md](SCANNER.md) if batch processing is needed

---

## Documentation Maintenance

### Updating Documentation
- **STATUS_LIFECYCLE.md** - Update when adding new statuses or changing intended behavior
- **STATUS_LIFECYCLE_REFACTORING.md** - Update as gaps are closed, mark sections as "✅ Implemented"
- **STATUS_LIFECYCLE_TODO.md** - Check off tasks as completed, add new tasks as discovered
- **ADMIN_TOOLS.md** - Update when adding new admin commands or filters
- **SCANNER.md** - Update when adding new processor types or patterns

### Documentation Review
Documentation should be reviewed:
- **Quarterly** - General review for accuracy and completeness
- **Per sprint** - Update TODO list and refactoring doc with progress
- **Before releases** - Ensure all new features are documented
- **After major refactorings** - Update all affected docs

### Documentation Standards
- Use markdown with GitHub-flavored syntax
- Include code examples with syntax highlighting
- Add diagrams for complex flows (ASCII art is fine)
- Link between documents using relative paths
- Keep line length reasonable (~100 chars) for readability
- Use clear headings for easy navigation

---

## Contributing to Documentation

### Adding New Documentation
1. Create the document in `docs/` directory
2. Add entry to this README under appropriate section
3. Add cross-references from related documents
4. Update the documentation hierarchy diagram

### Documentation Templates
Follow these patterns for consistency:

**Operational Guides** (like STATUS_LIFECYCLE.md):
- Overview and core concepts
- Detailed reference (schemas, APIs, etc.)
- Examples and use cases
- Best practices
- Troubleshooting
- Related documentation links

**Design Documents** (like STATUS_LIFECYCLE_REFACTORING.md):
- Executive summary
- Problem statement
- Current state analysis
- Proposed solution
- Implementation plan
- Testing strategy
- Migration path
- References

**Task Lists** (like STATUS_LIFECYCLE_TODO.md):
- Overview and context
- Pre-requisites
- Phase breakdowns
- Detailed task checklists
- Estimated timelines
- Success metrics

---

## External Resources

- [Project Repository](https://github.com/tendant/simple-content)
- [Issue Tracker](https://github.com/tendant/simple-content/issues)
- [Changelog](../CHANGELOG.md) (if exists)

---

## Questions or Feedback?

- **For documentation issues:** Open an issue on GitHub
- **For technical questions:** See [STATUS_LIFECYCLE.md § Troubleshooting](STATUS_LIFECYCLE.md#troubleshooting)
- **For feature requests:** Check [STATUS_LIFECYCLE_REFACTORING.md](STATUS_LIFECYCLE_REFACTORING.md) first, then open an issue

---

*Last updated: 2025-10-01*
