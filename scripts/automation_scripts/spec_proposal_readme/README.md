# Spec Proposal Sorter

### How to use:

```
python3 scripts/automation_scripts/spec_proposal_sorter.py
```

# Results:

A directory containing the below hierarchy

# Spec Hierarchy

### Overview
This folder contains specs that are organized by inheritance hierarchy, starting from the root spec (with no dependencies) and increasing in number.

### Purpose
The purpose of this structure is to simplify the process of creating new specs and aid automation scripts. By grouping related specs together, it is easier to manage and maintain them.

### Folder Structure
Each folder contains a maximum of 5 specs to ensure that the size of each transaction does not exceed the maximum limit. The hierarchy of specs within each folder reflects their inheritance relationship.