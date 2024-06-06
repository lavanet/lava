name: 🐛 Bug report
description: Create a report to help us squash bugs!
title: "[Bug]: "
labels: ["T:Bug"]

body:
  - type: markdown
    attributes:
      value: |
        Thanks for taking the time to fill out this bug report!
        Before smashing the submit button please review the template.

  - type: checkboxes
    attributes:
      label: Is there an existing issue for this?
      description: Please search existing issues to avoid creating duplicates.
      options:
        - label: I have searched the existing issues
          required: true

  - type: textarea
    id: what-happened
    attributes:
      label: What happened?
      description: Also tell us, what did you expect to happen?
      placeholder: Tell us what you see!
      value: "A bug happened!"
    validations:
      required: true

  - type: input
    attributes:
      label: Lava Version
      description: If applicable, specify the binary and the version you're using
      placeholder: lavad v0.46, lavap v0.47, main, etc.
    validations:
      required: true

  - type: textarea
    id: reproduce
    attributes:
      label: How to reproduce?
      description: If applicable could you describe how we could reproduce the bug
      placeholder: Tell us what how to reproduce the bug!
    validations:
      required: false