## Limitations and boundaries
Everest product has different dependencies and right now we have the following list

- Kubernetes version
- Operator Lifecycle Manager version
- Versions of upstream and monitoring operators (PXC operator, PSMDB operator, PG operator, and Victoria Metrics Operator)
- PMM version

While working on a feature or improvement you should keep in mind the dependencies of external components.

## Kubernetes versioning and limitations

1. We should ensure that Everest supports the same versions of Kubernetes clusters as upstream operators support.
2. Kubernetes has EOL dates. However, the EOL date of Kubernetes for Everest is the latest version taken across three cloud providers (AWS, GCP, Azure).
3. Using Kubernetes features should not limit us to any version of Kubernetes (E.g. Validation rules are available from 1.25, however, we still need to support 1.24 and in that case, we should wait until 1.24 EOL)
4. We should keep our operator in Namespaced mode as long as we can. 


## Best practices we follow

We follow these best practices

- [Go best practices](./go_best_practices.md)
- [Operator best practices](./operator_best_practice.md)

## Team principles
1. **We will not ship untested code.** All the code should be tested before we ship it!

2. **Stable Productivity**
    * We will not allow our project to become unstable in productivity. We know that the system has become unstable when every task becomes more complicated and takes more time.
    * We want to nurture a culture of continuous improvement rather than undertaking full system redesigns merely for the sake of change. We shall focus on making incremental enhancements that not only deliver tangible value but also maintain the team's productivity.
3. **Inexpensive Adaptability.** Software should be easy to change, that should be the first priority. A system that works but can’t be changed becomes obsolete. A system that doesn’t work but can be easily changed, can be easily fixed.
4. **Continuous Improvement.** Everything should get better with time.
5. **Fearless Competence.** Don’t be afraid to change the code. Testing allows you to make fearless changes.
6. **Bugs shall not pass!** We will make sure the code is tested before it reaches QA, to speed up this process. We implement bulletproof solutions where all corner cases should be handled
7. **We cover for each other.** We will behave like a team, we have to work together and code with each other. In case someone can’t work another can take over.
8. We follow [80/20 rule](https://en.wikipedia.org/wiki/Pareto_principle) meaning that 80% of users will benefit from 20% of features.
9. We follow [KISS](https://en.wikipedia.org/wiki/KISS_principle), [YAGNI](https://en.wikipedia.org/wiki/You_aren%27t_gonna_need_it) and Less is More. Basically, it means that we try to go with the simplest and minimalistic solution for a problem and we will improve it in later iterations. Followed by the 80/20 principle will help us build software that is simple enough but solves a user's problem and we can improve it later according to user feedback. 
10. **We do not optimize something prematurely**
11. **Predictability over performance**. This means having a service that will serve their data within the 95th percentile of latency the majority of the time, rather than having extremely high performance that is not predictable. Similarly, all operations should be predictable or otherwise explainable to users.

## Proposing to change something 

It's always great to get proposals on how we can improve the design or other parts of the system yet before building PoC or something else please write a design document or written narrative to support your idea. You can use [short proposal template](./short_proposal_template.md) or [full proposal template](./full_proposal_template.md).
