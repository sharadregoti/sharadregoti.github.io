+++
title = 'What I Have Understood About DevOps, As An Engineer With Just 2yrs Of Experience'
date = 2024-01-21T10:39:08+05:30
draft = false
+++

When I first heard the term DevOps, What I understood from by peers is that, 

> “The process of deploying application in any environment (dev/QA/prod) is called DevOps. It’s just another synonym for operations.”
> 

As a novice programmer I was like,

> “Okay, Cool !!!, It’s just another buzz world floating around in the IT industry”
> 

People who have some idea about DevOps, know how wrong I am!😆

But after spending some time in the IT industry & seeing job recruitment posts with designations such as **requirement of AI-Ops, ML-Ops, Data-Ops, Fin-Ops…. X-Ops engineer**, I was like wait a minute, DevOps has to be more than just deployment.

https://tenor.com/view/wait-what-meme-wait-a-minute-gif-14484132

When I googled DevOps, Here are a few definitions, I found on the internet

> “DevOps is the combination of cultural philosophies, practices, and tools that increases an organization’s ability to deliver applications and services at high velocity.”
> 

> “It is an intersection between development & operations.”
> 

> “You take a process that is being done manually & automate it.”
> 

Here’s an interesting perspective of DevOps.

https://www.youtube.com/watch?v=yczwWzkbFAQ

The term in general is so loosely defined that it is difficult to understand, especially when you don’t have much historical context of software development that led to this revolution.

<aside>
💡 But these definitions made be think

- Aren’t we deploying application for the last 20 years ?
- What really led to the need of automation in the first place?
- What are the inherent problems that we are trying to solve with DevOps?
- Which part of dev is not in DevOps & which part of operations is not DevOps, Why there was a need between these?
- Can’t we call it a day just by writing some CI/CD pipelines?
- When can I say that I have a successful DevOps practice at my organization?
</aside>

Many more such questions came to my mind. So to really understand DevOps, we need some historical context of how software delivery process has always been carried out.

# Understanding The Software Release Process & Its History

📌 Following are the stages involved in a typical software development process.

!https://raw.githubusercontent.com/sharadregoti/sharadregoti.github.io/main/images/2022-07-24-09-what-is-devops/blog.drawio.svg

Our software development process hasn’t changed much over the last 3 decades. Software teams are usually comprised of development teams (who write code) & operation teams (who deploy & maintain applications). Both these teams are responsible for software releases.

With the above team structure in mind & the way they operate, we can get some historical context.

During the old days software releases used to be less frequent as the requirements hardly changed ever. Systems were not designed to change fast, after deployment. They are intended to be there and be stable.

The only thing that had to be done post application deployment is general maintenance. You know things like upgrading the OS and packages that the system requires.

Infrastructure was not as readily accessible as it is now. As a result of that the operation team was responsible for static capacity planning, provisioning & maintenance of the same. This made the development team and the operations team move at more or less the same rate. 

But If you observe carefully, the way we used to do IT is very silo oriented, because you hired people who were specialist in distinct areas. There was not much co-ordination required between teams.

This approach worked great during that time as the frequency of deployment was low. It was rare to see more than one release per month.

Processes & practices were established, and teams adapted to this type of development approach.

But as time passed & requirements started to evolve rapidly. Problems started to appear in the software delivery process

# Problems That Led To The DevOps Revolution

## 1) Changing Requirements

Suddenlty there was a need to deploy frequently. The release cycles reduced from months to weeks. But as the idea of changing requirement was not part of the calculus, trying to apply the legacy way of doing things to new changes didn’t always work out.

The operations team found it hard to deploy frequently while ensure stability of the system.

The process & practices started to fail, mainly because of the lack of co-ordination between the teams & tools that were being used (We will explore this in more detail later).

Operation team tried to adapt with the change by writing some scripts & changing processes to get some relief.

But as software system got more complex with the involvement of containerization, microservices, auto scaling & what not, it just got out of the hand. A lot of roadblocks started showing up in the software delivery process.

!https://raw.githubusercontent.com/sharadregoti/sharadregoti.github.io/main/images/2022-07-24-09-what-is-devops/blog-hurdles.svg

## 2) Friction Between Teams

The real problem is with the people who are developing & maintaining the software.

Any organization's software team has a straightforward objective:

> Deliver high quality software faster.
> 

And everyone in the team is working together to achieve that common objective. However, in practice, software teams don't operate as a unified entity since each team member has a different incentive to work that takes priority over the common objective.

Let’s take a look at incentives of individuals in a team

- Development Team
    
    Incentive: Develop new features faster
    
- QA Team
    
    Incentive: Ensure all test scenarios are covered (time consuming activity)
    
- Security Team
    
    Incentive: Ensure security best practices are adhered (time consuming activity)
    
- Operations Team
    
    Incentive: Ensure stability of application running in production (time consuming activity)
    

Because of this incentives

- **Development and operations have contradictory goals:** This cause a great wall to form between them, introduces friction & slows down the entire release process.
    
    !https://raw.githubusercontent.com/sharadregoti/sharadregoti.github.io/main/images/2022-07-24-09-what-is-devops/blog-Wall.svg
    
- Teams are working in silos & communication is not happening properly, which leads to problems from both the ends
    - Developers
        - Deployment guide is not well documented
        - Doesn’t consider where the app is getting deployed
    - Operations
        - Don’t know how the app works
        - If something fails, need help of developer to figure it out
- Priority is given to completing their incentive & rest is not their problem
- This stretches release period from days to weeks to months

**As the team incentives become greater than the common objective, this results into deadlines, production failures, bad user experience etc.** All of this ultimately impacts the business.

So to remove the roadblocks,

# **DevOps was introduced as a solution for**

> “Delivering high quality software, by making the process fast & ensuring stability.”
> 

!https://raw.githubusercontent.com/sharadregoti/sharadregoti.github.io/main/images/2022-07-24-09-what-is-devops/devops-toolchain.svg

The role of DevOps team here is to identify such roadblocks in software development lifecycle & try to overcome by introducing some kind of automation. This could be via. tools & scripts or processes & practices that will lead to an improved development & deployment experience.

So essentially, DevOps is trying to remove the roadblocks by introducing automation & streamlining the software delivery process. That is why we have CI & CD processes at the center of DevOps.

To conclude, I would like quote the meaning of DevOps from [@Patrick Dubois](https://twitter.com/patrickdebois) (the person who coined the term DevOps)

> “Doing anything & everything to overcome the friction created by silos… All the rest is plain engineering”
> 

---

That’s it in this blog post, If you liked this blog post. You can also checkout my YouTube channel where we talk about M**icroservices, Cloud & Kubernetes,** you can check it out here 👉[Youtube](https://l.linklyhq.com/l/KhmP)

If you have any questions you can reach me on Twitter at [@SharadRegoti](https://twitter.com/SharadRegoti)