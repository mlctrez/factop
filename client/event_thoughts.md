# UDP Event Expansion Planning

Current player.lua events already emitting UDP messages:
`join`, `leave`, `death`, `respawn`, `move`

## High-Value Events for Plugin Development

### Combat & Entity Lifecycle

**on_entity_died** — Fires when any entity is destroyed. Useful for tracking
base defense breaches, turret kills, and biter wave outcomes. The event
provides `cause` and `entity` fields so plugins could build kill feeds,
damage heatmaps, or alert on structure losses.

**on_entity_damaged** — Fires on each damage tick. High volume, so would
need filtering (e.g. only player-owned structures, or only above a damage
threshold). Could power real-time base health dashboards or trigger
automated repair dispatching.

**on_post_entity_died** — Fires after death processing completes and
includes `ghost` if one was placed. Useful for auto-rebuild plugins that
want to know what ghost to target.

### Construction & Deconstruction

**on_built_entity** — Player placed an entity. Enables build logging,
construction analytics, or plugins that react to specific placements
(e.g. auto-configure new train stops, validate blueprint placement).

**on_player_mined_entity** — Player removed an entity. Paired with
on_built_entity, gives full construction/deconstruction tracking.

**on_robot_built_entity** / **on_robot_mined_entity** — Same as above but
for construction robots. Important for tracking automated construction
activity separately from manual player actions.

### Research & Progression

**on_research_finished** — A technology was completed. Plugins could
announce unlocks, trigger automated builds of newly available entities,
or track progression speed across playthroughs.

**on_research_started** — Research queue changed. Useful for coordination
plugins in multiplayer that want to notify players about research choices.

### Chat & Console

**on_console_chat** — Player sent a chat message. Enables chat bridges
to external systems (Discord, Slack), chat command plugins, or chat
logging/moderation tools.

**on_console_command** — A console command was executed. Useful for
audit logging of admin actions.

### Trains & Logistics

**on_train_changed_state** — Train arrived, departed, waiting at signal,
etc. High value for logistics monitoring plugins, train network
dashboards, or deadlock detection.

**on_train_schedule_changed** — Schedule was modified. Useful for
tracking train network configuration changes.

### Surface & World

**on_player_changed_surface** — Player moved between surfaces (e.g.
nauvis to space platform). Important for multi-surface tracking,
especially with Space Age expansion content.

**on_surface_created** / **on_surface_deleted** — New surfaces added or
removed. Relevant for plugins managing cross-surface logistics or
monitoring space platform creation.

**on_chunk_generated** — New map chunks revealed. Could power map
expansion tracking or resource survey plugins.

### Resource Management

**on_resource_depleted** — A resource patch tile ran out. Useful for
resource monitoring plugins that alert when patches are running low.

**on_picked_up_item** / **on_player_dropped_item** — Item pickup and
drop events. Could enable item flow tracking or loot logging.

### Rocket & Space

**on_rocket_launched** — A rocket was launched. Milestone tracking,
launch counters, or triggering post-launch automation.

**on_cargo_pod_delivered_cargo** — Cargo arrived at destination. Useful
for space logistics monitoring.

## Lower Priority but Worth Noting

**on_player_crafted_item** — Manual crafting completed. Niche but useful
for tracking player self-sufficiency vs factory output.

**on_player_driving_changed_state** — Player entered/exited a vehicle.
Could be interesting for vehicle usage analytics.

**on_sector_scanned** — Radar scanned a new sector. Map coverage tracking.

**on_player_built_tile** / **on_player_mined_tile** — Tile placement and
removal. Already have tile RCON commands, but event-driven tracking could
complement them.

## Implementation Considerations

- High-frequency events (on_entity_damaged, on_tick) need filtering or
  throttling to avoid flooding the UDP bridge.
- Each new event should follow the established pattern: colon-separated
  wire format, log_event through UDP, and a corresponding Parse function
  in the appropriate client package.
- Events not tied to a specific player (on_entity_died from biters,
  on_train_changed_state) would need a different module than player.lua.
  Consider an `event.lua` module for non-player events.
- Group related events to keep the number of new modules manageable.
  Combat events could share a module, train events another.
