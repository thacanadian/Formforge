# FormForge 1.7 UI Experience

## Design direction

FormForge now uses a focused retro-terminal interface: deep neutral surfaces, restrained amber highlights, thin technical borders, and generous negative space. The visual language is intentionally quieter than a full HUD. Important actions glow; background decoration does not compete with content.

## Default information architecture

The focused desktop navigation contains:

- Dashboard
- AI Coach
- Workouts
- Nutrition
- Progress
- More

The mobile bottom bar contains the five most-used destinations. More is the complete feature directory and retains access to Coaching Team, FormForge Agent, Health + Recovery, Habits, Weekly Check-In, Community, Marketplace, Mobile setup, Appearance + Units, Security, Profile + Data, and authorized administrator tools.

Users can still choose full navigation from Appearance. Administrators can preview the member experience from More without losing administrator permissions.

## Dashboard

The default command center intentionally shows only four primary modules:

1. Today’s Workout — plan details and the primary Start Workout action.
2. Recovery Score — score, status, and recent recovery inputs.
3. AI Coach — one contextual recommendation and a direct adjustment action.
4. Nutrition Summary — calories and macro progress.

Supporting tools remain one action away rather than appearing as additional dashboard panels.

## Motion

Motion is used for orientation and feedback:

- sections reveal as they enter the viewport;
- pages transition when navigation changes;
- interactive cards and buttons use subtle depth and glow;
- a thin progress indicator reflects page scroll position;
- active workout controls provide immediate completion feedback.

When `prefers-reduced-motion: reduce` is enabled, nonessential animation, smooth scrolling, and transforms are disabled.

## Responsive behavior

Desktop uses a persistent navigation rail and a two-column command center. Tablet collapses spacing and card proportions. Phone layouts stack the four primary cards and use a fixed bottom navigation bar with touch-sized controls. Secondary features remain in More rather than crowding the bottom navigation.

## Functional preservation

The redesign changes presentation and navigation, not data models or permissions. Existing API-backed workflows, account roles, AI modes, tracking screens, security tools, and administrator features remain available.
