# TODO list
- [x] get supabase url and put in env
- [x] find which error i am having

- [x]  New player joined got 0 points and at top 1
> [!note]
> player score in **room_player** table is `null`

- [ ] player score not increased after submitting

- [ ] Add joined date to room_player

- [ ] if player left, they are removed from the **room_player** table
> [!note]
> we don't want this, player who joined will persist in the room, even if they left
> maybe we want to have a state of user in the room
>

- [ ] implement state for room_player
> | state   | meaning                        |
> |---      |---                             |
> | PRESENT | Player is currently in the room|
> | DISCONNECTED | Player got disconnected from the room |
> | LEFT | Player has been diconnected for too long (5 minutes)|
> | COMPLETED | Player with `PRESENT` state will be changed to COMPLETED at the end |

- [ ] implement `DISCONNECTED` to `LEFT`
> [!note]
> implementation: add a job with 5 minutes delay to the message queue(rabbitmq, redis)
>

- [-] judge system (no need web interface)
> [!note]
> judge system only job is to `compile` & `run` the submitted code and pass the result to a message queue (rabbitmq, redis)
>

- [x] Fix this bug
> [!bug]
> Bug on submitting solution
> Submitting too quick causes unlock an already unlocked mutex
-> Fixed: Forgot to set container state to idle after done executing code

- [ ] webrtc for meeting
- [ ] media server to get user transcript and use LLM to summarize

> [!note]
> We will make 2 images, one for the judge system (including code), and one for webrtc + coding battle, as opposed to the
> prototype design where we make one big image (inspired by [judge0][https://github.com/judge0/judge0]) that contains  both
> main service and the judge environment (hard to scale).


> [!note]
> xcodeengine in-depth:
> main.go initialize workerpool 
> workerpool init a **ContainerManager**
> **ContainerManager** has `docker client`, list of current containers (map string -> ContainerInfo), maxWorkers, a mutex, memoryLimit, cpunanoLimit
> Container Manager will initialize a number of containers running using the maxWorkers

# Flow
User Submit Code -> SubmitSolutionHandler -> Sanitize code + Worker Pool -> Worker Pool with ContainerManager 



