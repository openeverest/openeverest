# Working with Git and GitHub

The golden rule

Commit and push to the feature branches daily. Use pull requests (in the draft state until they are ready for review) to show WIP. Saying "I'm coding feature X" on the daily call without pushed commits is not okay.

## Preparation

Make sure your `git user.name` is set, and `user.email` is set to your corporate (if you are Perconian) or personal email, i.e. john.doe@percona.com.

Use topic branches in the same repository instead of GitHub forks if you have such permissions.

Name topic branches like this: `EVEREST-1234-short-description`. Always start with `EVEREST-XXXX`. Never omit a short description. Separate words with dashes, not underscore. Use only lowercase letters.

If you are community contributor and no ticket exists, either create one or just use `short-description` style for your forked repo branch.

## Commit rules

Every commit message should use this commit template:
```
EVEREST-XXXX Short summary up to 50 characters.

Optional 72-character wrapped longer description.
```

There should be a blank line between short summary and longer description. Final dot in the title is optional – if you hate it, drop it :)

While addressing review comments push the requested changes as a new commit instead of amending the original commit and force pushing to GitHub.

## Pull Request rules

1. Before Creating or updating PR always run linters and tests locally.
2. Every PR should follow the template of a commit message.
3. If PR provides a prototype or a proof-of-concept, PR should be draft so it is not accidentally merged.
4. Start PR as draft ASAP to let everyone see the job you are doing.
5. Check-list of things to do prior to submitting a PR for review:
    - [x] All tests in the PR pass.
    - [x] Clean up your code removing unnecessary tokens, code which is commented out, etc.
    - [x] All required lint rules pass.
    - [x] New tests are written if it's possible for new changes including unit and integration tests.
6. Rules for merging a PR:
    - [x] All required checks must pass.
    - [x] The reviewers (minimum 2) are set as makes sense for the code (DevOps, BE, FE, QA), and all need to approve the code after review.
    - [x] If change requests were made, all changes need to be resolved and re-approved by the original reviewers.
    - [x] If all conditions for a successful merge are met (checks pass, code reviewed and approved, changes resolved and re-approved), ONLY then the PR can be merged:
         * If an author has write permissions. An author should merge the PR.
         * If an author doesn't have write permissions, then author should ask someone with write permissions to merge.
         * If required checks fail for a confusing reason or some non-required checks fail for a long time, ping the appropriate dev channel.
    - [x] A PR should be merged with the "Squash and Merge" button (the code owner should set this as the only option in the repository settings).
    - [x] Never leave PRs fully approved, unassigned, and not merged.
    - [x] Tasks with several PRs should be merged with minimum time interval between the merges.
7. If you need to re-request a review from someone, please use GitHub feature as displayed in Figure 1 below (use case: I made some changes after the PR was reviewed so I want to request another review from the same reviewer). Ping reviewer if it's not reviewed in 1 working day.
8. If you open a Pull Request to fix a Bug - put an explanation in the Long Description about the Bug. It may contain information about how it affects the component. Also, it's good to have short references on how the problem is solved.


[Check GitHub PR review documentation](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/requesting-a-pull-request-review).

Figure 1. Request another review from a reviewer.
![](https://docs.github.com/assets/cb-4714/images/help/pull_requests/request-re-review.png "")

## Reviewing Pull requests

1. Ensure the description of PR communicates the problem that it solves and there's needed information to get context (ticket, description)
2. Ensure unit/integration tests are added
3. Be polite, elaborate when you're proposing a different solution, or ask questions. E.g. Why did you decide to use x instead of y?
4. Explain clearly why PR needs to address changes (security concerns, code quality, lack of tests coverage, any other reason)
5. Be humble. Learn from others
