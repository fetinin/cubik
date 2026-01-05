# Animation Storage and Loading Feature - Implementation Plan

## Current System Understanding

### Backend (Go)
- **Animation Structure**: Frames are `[][]Color` (array of frames, each frame is 100 Color structs for 20×5 matrix)
- **Current API Endpoints**:
  - `GET /api/devices` - discover devices
  - `POST /animation/start` - start animation with frames
  - `POST /animation/stop` - stop animation
- **Handler Pattern**: Uses ogen-generated code, validates inputs, returns discriminated union responses
- **No Database**: Currently no persistence layer

### Frontend (SvelteKit + TypeScript)
- **State Management**: Svelte stores in `/front/src/lib/state/editor.ts`
- **Frame Storage**: Frames stored as `Frame[]` with packed RGB integers (0xRRGGBB)
- **API Client**: Auto-generated TypeScript client from OpenAPI spec
- **UI Components**:
  - MatrixGrid - interactive pixel editor
  - AnimationPreview - 1 FPS preview player
  - FramesPanel - frame management
  - Apply section - send to device buttons

---

## Proposed Changes Overview

### 1. Backend API Changes

#### New Endpoints (spec.yml additions):

**POST /animation/save**
- **Purpose**: Save current animation frames to database
- **Request**: `{ device_id: string, name: string, frames: AnimationFrame[] }`
- **Response**: `{ id: string, message: string, animation: SavedAnimation }`
- **Handler**: `handler.go::SaveAnimation()`

**GET /animation/list/{device_id}**
- **Purpose**: Get all saved animations for a device
- **Request**: Device ID in path parameter
- **Response**: `{ animations: SavedAnimation[] }`
- **Handler**: `handler.go::ListAnimations()`

**GET /animation/{id}**
- **Purpose**: Load a specific saved animation by ID
- **Request**: Animation ID in path parameter
- **Response**: `{ animation: SavedAnimation }`
- **Handler**: `handler.go::GetAnimation()`

**PUT /animation/{id}**
- **Purpose**: Overwrite existing animation
- **Request**: `{ name: string, frames: AnimationFrame[] }`
- **Response**: `{ message: string, animation: SavedAnimation }`
- **Handler**: `handler.go::UpdateAnimation()`

**DELETE /animation/{id}**
- **Purpose**: Delete saved animation
- **Request**: Animation ID in path parameter
- **Response**: `{ message: string }`
- **Handler**: `handler.go::DeleteAnimation()`

#### New OpenAPI Schemas:

```yaml
SavedAnimation:
  type: object
  properties:
    id: string (UUID)
    device_id: string
    name: string
    frames: AnimationFrame[]
    created_at: string (ISO 8601)
    updated_at: string (ISO 8601)
```

### 2. Backend Storage Layer

#### New File: `storage.go`

**Database Setup**:
- SQLite database file: `animations.db`
- Table: `saved_animations`
- Columns: `id, device_id, name, frames_json, created_at, updated_at`

**Functions**:
- `InitDB() error` - Initialize SQLite connection and create schema
- `SaveAnimation(deviceID, name string, frames [][]Color) (*SavedAnimation, error)`
- `GetAnimation(id string) (*SavedAnimation, error)`
- `ListAnimationsByDevice(deviceID string) ([]*SavedAnimation, error)`
- `UpdateAnimation(id, name string, frames [][]Color) (*SavedAnimation, error)`
- `DeleteAnimation(id string) error`

**JSON Serialization**:
- Store frames as JSON blob: `[][]{"r": int, "g": int, "b": int}`
- Convert between `[][]Color` and JSON on save/load

#### Changes to `handler.go`:

Add 5 new handler methods implementing the new endpoints:
- `SaveAnimation(ctx, req) -> SaveAnimationRes`
- `ListAnimations(ctx, req) -> ListAnimationsRes`
- `GetAnimation(ctx, req) -> GetAnimationRes`
- `UpdateAnimation(ctx, req) -> UpdateAnimationRes`
- `DeleteAnimation(ctx, req) -> DeleteAnimationRes`

Each follows existing validation → business logic → response pattern.

#### Changes to `main.go`:

Add database initialization:
```go
func main() {
    if err := InitDB(); err != nil {
        log.Fatal("Failed to init database:", err)
    }
    defer CloseDB()
    // ... existing code
}
```

### 3. Frontend Changes

#### API Client Regeneration:
Run `cd front && bun run generate-api` after updating `spec.yml`

#### New State (editor.ts):

```typescript
// Add to EditorState:
savedAnimations: Writable<SavedAnimation[]>
currentAnimationId: Writable<string | null>  // Track which saved animation is loaded
```

#### New Component: `SavedAnimationsList.svelte`

- Lists saved animations for selected device
- Shows animation names with preview thumbnails
- Click to load animation
- Delete button for each animation

#### Changes to Main Page (`+page.svelte`):

**New Buttons in Apply Section**:
1. **"Save Animation"** button
   - Opens modal/prompt for animation name
   - Calls `saveAnimation()` function
   - If `currentAnimationId` is set, offers to overwrite

2. **"Load Animation"** button
   - Opens saved animations list panel
   - Clicking an animation loads frames into editor
   - Sets `currentAnimationId` for tracking

**New Functions**:
```typescript
async function saveAnimation(name: string, overwrite: boolean) {
    const deviceLocation = get(selectedDevice)?.location;
    const framesList = get(frames);

    if (overwrite && currentAnimationId) {
        await api.updateAnimation(currentAnimationId, { name, frames });
    } else {
        const result = await api.saveAnimation({ device_id, name, frames });
        currentAnimationId.set(result.id);
    }
}

async function loadAnimation(id: string) {
    const animation = await api.getAnimation(id);
    frames.set(animation.frames);  // Load frames into editor
    currentAnimationId.set(id);     // Track which animation is loaded
}

async function deleteAnimation(id: string) {
    await api.deleteAnimation(id);
    await refreshAnimationsList();
}
```

#### Update `mock.ts`:

Add wrapper functions for new endpoints to match existing pattern.

---

## Critical Files to Modify

### Backend:
1. **spec.yml** - Add 5 new endpoints + SavedAnimation schema
2. **storage.go** (NEW) - SQLite database layer
3. **handler.go** - Add 5 new handler methods
4. **main.go** - Initialize database on startup
5. **go.mod** - Add SQLite driver dependency (`github.com/mattn/go-sqlite3`)

### Frontend:
1. **front/src/lib/state/editor.ts** - Add savedAnimations state
2. **front/src/routes/+page.svelte** - Add save/load buttons and logic
3. **front/src/lib/components/SavedAnimationsList.svelte** (NEW) - Animation browser
4. **front/src/lib/api/mock.ts** - Add wrapper functions for new endpoints
5. **Regenerate**: `front/src/api/generated/*` (via `bun run generate-api`)

---

## Implementation Steps

1. **Backend Foundation**
   - Add SQLite dependency
   - Create storage.go with database schema
   - Implement CRUD functions

2. **API Specification**
   - Update spec.yml with new endpoints
   - Define SavedAnimation schema
   - Run `go generate ./...` to regenerate Go code

3. **Backend Handlers**
   - Implement 5 new handler methods
   - Add validation logic
   - Test with curl/Postman

4. **Frontend API Client**
   - Run `bun run generate-api` to get new TypeScript types
   - Update mock.ts wrappers

5. **Frontend UI**
   - Add state for saved animations
   - Create SavedAnimationsList component
   - Add save/load buttons to main page
   - Implement save/load/delete logic

6. **Integration Testing**
   - Test full flow: draw → save → load → modify → overwrite

---

## User Requirements (Confirmed)

1. ✅ **Storage Scope**: Per device - animations stored with device_id, filtered by device
2. ✅ **Save Behavior**: Smart save - detect if editing loaded animation, offer update vs save as new
3. ✅ **UI Placement**: Modal dialog - clicking "Load Animation" opens modal with list
4. ✅ **Metadata**: Minimal - only name, device_id, frames, created_at, updated_at (no description/thumbnail)

---

## Detailed Implementation Plan

### Phase 1: Backend Database Layer

#### 1.1 Add SQLite Dependency (go.mod)
```bash
go get github.com/mattn/go-sqlite3
```

#### 1.2 Create storage.go

**Database Schema**:
```sql
CREATE TABLE IF NOT EXISTS saved_animations (
    id TEXT PRIMARY KEY,           -- UUID
    device_id TEXT NOT NULL,       -- Device location (yeelight://IP:PORT)
    name TEXT NOT NULL,
    frames_json TEXT NOT NULL,     -- JSON array of frames
    created_at TEXT NOT NULL,      -- ISO 8601
    updated_at TEXT NOT NULL       -- ISO 8601
);
CREATE INDEX idx_device_id ON saved_animations(device_id);
```

**Core Functions**:
- `InitDB() error` - Open SQLite connection, create schema
- `CloseDB()` - Close database connection
- `SaveAnimation(deviceID, name string, frames [][]Color) (*SavedAnimation, error)`
  - Generate UUID
  - Serialize frames to JSON
  - INSERT into database
- `GetAnimation(id string) (*SavedAnimation, error)`
  - SELECT by id
  - Deserialize frames from JSON
- `ListAnimationsByDevice(deviceID string) ([]*SavedAnimation, error)`
  - SELECT WHERE device_id = ?
  - Order by updated_at DESC
- `UpdateAnimation(id, name string, frames [][]Color) (*SavedAnimation, error)`
  - UPDATE name, frames_json, updated_at WHERE id = ?
- `DeleteAnimation(id string) error`
  - DELETE WHERE id = ?

**JSON Serialization**:
```go
// Serialize [][]Color -> JSON string
type FrameJSON struct {
    R uint8 `json:"r"`
    G uint8 `json:"g"`
    B uint8 `json:"b"`
}

func serializeFrames(frames [][]Color) (string, error) {
    // Convert [][]Color to [][][]FrameJSON
    // json.Marshal()
}

func deserializeFrames(jsonStr string) ([][]Color, error) {
    // json.Unmarshal()
    // Convert back to [][]Color
}
```

### Phase 2: Backend API Specification

#### 2.1 Update spec.yml

**Add SavedAnimation Schema**:
```yaml
SavedAnimation:
  type: object
  required: [id, device_id, name, frames, created_at, updated_at]
  properties:
    id:
      type: string
      format: uuid
    device_id:
      type: string
      pattern: '^yeelight://[0-9.]+:[0-9]+$'
    name:
      type: string
      minLength: 1
      maxLength: 100
    frames:
      type: array
      items:
        $ref: '#/components/schemas/AnimationFrame'
    created_at:
      type: string
      format: date-time
    updated_at:
      type: string
      format: date-time
```

**Add 5 New Endpoints**:

1. **POST /animation/save**
   ```yaml
   requestBody:
     required: true
     content:
       application/json:
         schema:
           type: object
           required: [device_id, name, frames]
           properties:
             device_id: string (yeelight://...)
             name: string
             frames: AnimationFrame[]
   responses:
     200:
       schema:
         type: object
         properties:
           id: string
           message: string
           animation: SavedAnimation
     400: BadRequest
     500: InternalServerError
   ```

2. **GET /animation/list/{device_id}**
   ```yaml
   parameters:
     - name: device_id
       in: path
       required: true
       schema:
         type: string
   responses:
     200:
       schema:
         type: object
         properties:
           animations: SavedAnimation[]
   ```

3. **GET /animation/{id}**
   ```yaml
   parameters:
     - name: id
       in: path
       required: true
   responses:
     200:
       schema:
         type: object
         properties:
           animation: SavedAnimation
     404: NotFound
   ```

4. **PUT /animation/{id}**
   ```yaml
   requestBody:
     schema:
       type: object
       required: [name, frames]
       properties:
         name: string
         frames: AnimationFrame[]
   responses:
     200:
       schema:
         properties:
           message: string
           animation: SavedAnimation
     404: NotFound
   ```

5. **DELETE /animation/{id}**
   ```yaml
   responses:
     200:
       schema:
         properties:
           message: string
     404: NotFound
   ```

After editing spec.yml, run: `go generate ./...`

### Phase 3: Backend Handlers

#### 3.1 Update handler.go

Add 5 new handler methods following existing patterns:

```go
func (h *APIHandler) SaveAnimation(ctx context.Context, req *api.SaveAnimationRequest) (api.SaveAnimationRes, error) {
    // Validate device_id format
    if !strings.HasPrefix(req.DeviceID, "yeelight://") {
        return &api.SaveAnimationBadRequest{Error: "invalid device_id format"}, nil
    }

    // Validate frames not empty
    if len(req.Frames) == 0 {
        return &api.SaveAnimationBadRequest{Error: "frames cannot be empty"}, nil
    }

    // Validate name
    if req.Name == "" {
        return &api.SaveAnimationBadRequest{Error: "name is required"}, nil
    }

    // Convert API frames to internal Color format
    frames := make([][]Color, len(req.Frames))
    for i, apiFrame := range req.Frames {
        frames[i] = ConvertAPIFrameToColors(apiFrame)
    }

    // Save to database
    animation, err := SaveAnimation(req.DeviceID, req.Name, frames)
    if err != nil {
        return &api.SaveAnimationInternalServerError{
            Error: fmt.Sprintf("failed to save animation: %v", err),
        }, nil
    }

    // Convert back to API format
    apiAnimation := convertToAPIAnimation(animation)

    return &api.SaveAnimationResponse{
        ID: animation.ID,
        Message: "Animation saved successfully",
        Animation: apiAnimation,
    }, nil
}

func (h *APIHandler) ListAnimations(ctx context.Context, req *api.ListAnimationsRequest) (api.ListAnimationsRes, error) {
    // Get from database
    animations, err := ListAnimationsByDevice(req.DeviceID)
    if err != nil {
        return &api.ListAnimationsInternalServerError{...}, nil
    }

    // Convert to API format
    apiAnimations := make([]api.SavedAnimation, len(animations))
    for i, anim := range animations {
        apiAnimations[i] = convertToAPIAnimation(anim)
    }

    return &api.ListAnimationsResponse{Animations: apiAnimations}, nil
}

func (h *APIHandler) GetAnimation(ctx context.Context, req *api.GetAnimationRequest) (api.GetAnimationRes, error) {
    animation, err := GetAnimation(req.ID)
    if err != nil {
        if err == ErrNotFound {
            return &api.GetAnimationNotFound{Error: "animation not found"}, nil
        }
        return &api.GetAnimationInternalServerError{...}, nil
    }

    return &api.GetAnimationResponse{
        Animation: convertToAPIAnimation(animation),
    }, nil
}

func (h *APIHandler) UpdateAnimation(ctx context.Context, req *api.UpdateAnimationRequest) (api.UpdateAnimationRes, error) {
    // Validate frames
    if len(req.Frames) == 0 {
        return &api.UpdateAnimationBadRequest{...}, nil
    }

    // Convert frames
    frames := make([][]Color, len(req.Frames))
    for i, apiFrame := range req.Frames {
        frames[i] = ConvertAPIFrameToColors(apiFrame)
    }

    // Update in database
    animation, err := UpdateAnimation(req.ID, req.Name, frames)
    if err != nil {
        if err == ErrNotFound {
            return &api.UpdateAnimationNotFound{...}, nil
        }
        return &api.UpdateAnimationInternalServerError{...}, nil
    }

    return &api.UpdateAnimationResponse{
        Message: "Animation updated successfully",
        Animation: convertToAPIAnimation(animation),
    }, nil
}

func (h *APIHandler) DeleteAnimation(ctx context.Context, req *api.DeleteAnimationRequest) (api.DeleteAnimationRes, error) {
    err := DeleteAnimation(req.ID)
    if err != nil {
        if err == ErrNotFound {
            return &api.DeleteAnimationNotFound{...}, nil
        }
        return &api.DeleteAnimationInternalServerError{...}, nil
    }

    return &api.DeleteAnimationResponse{
        Message: "Animation deleted successfully",
    }, nil
}

// Helper function
func convertToAPIAnimation(anim *SavedAnimation) api.SavedAnimation {
    // Convert [][]Color to [][]api.RGBPixel
    apiFrames := make([][]api.RGBPixel, len(anim.Frames))
    for i, frame := range anim.Frames {
        apiFrames[i] = make([]api.RGBPixel, len(frame))
        for j, color := range frame {
            apiFrames[i][j] = api.RGBPixel{
                R: int(color.R),
                G: int(color.G),
                B: int(color.B),
            }
        }
    }

    return api.SavedAnimation{
        ID: anim.ID,
        DeviceID: anim.DeviceID,
        Name: anim.Name,
        Frames: apiFrames,
        CreatedAt: anim.CreatedAt,
        UpdatedAt: anim.UpdatedAt,
    }
}
```

#### 3.2 Update main.go

Add database initialization:
```go
func main() {
    // Initialize database
    if err := InitDB(); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer CloseDB()

    // ... existing flag parsing and server/demo mode code
}
```

### Phase 4: Frontend API Client

#### 4.1 Regenerate TypeScript Client

After updating spec.yml and regenerating Go code:
```bash
cd front
bun run generate-api
```

This creates new TypeScript types and methods in `front/src/api/generated/`:
- `SavedAnimation` interface
- `api.saveAnimation()`
- `api.listAnimations()`
- `api.getAnimation()`
- `api.updateAnimation()`
- `api.deleteAnimation()`

#### 4.2 Update mock.ts

Add wrapper functions following existing pattern:

```typescript
export async function saveAnimation(
    deviceId: string,
    name: string,
    frames: number[][]
): Promise<SavedAnimation> {
    const toPixel = (packed: number): RGBPixel => ({
        r: (packed >> 16) & 0xff,
        g: (packed >> 8) & 0xff,
        b: packed & 0xff
    });

    const response = await api.saveAnimation({
        saveAnimationRequest: {
            device_id: deviceId,
            name,
            frames: frames.map((frame) => frame.map(toPixel))
        }
    });

    return response.animation;
}

export async function listAnimations(deviceId: string): Promise<SavedAnimation[]> {
    const response = await api.listAnimations({ deviceId });
    return response.animations;
}

export async function loadAnimation(id: string): Promise<SavedAnimation> {
    const response = await api.getAnimation({ id });
    return response.animation;
}

export async function updateAnimation(
    id: string,
    name: string,
    frames: number[][]
): Promise<SavedAnimation> {
    const toPixel = (packed: number): RGBPixel => ({...});

    const response = await api.updateAnimation({
        id,
        updateAnimationRequest: {
            name,
            frames: frames.map((frame) => frame.map(toPixel))
        }
    });

    return response.animation;
}

export async function deleteAnimation(id: string): Promise<void> {
    await api.deleteAnimation({ id });
}
```

### Phase 5: Frontend State Management

#### 5.1 Update editor.ts

Add new state fields:

```typescript
export type EditorState = {
    // ... existing fields

    // New fields for saved animations
    savedAnimations: Writable<SavedAnimation[]>;
    currentAnimationId: Writable<string | null>;  // ID of loaded animation
    showLoadModal: Writable<boolean>;
    showSaveModal: Writable<boolean>;
};

export function createEditorState(): EditorState {
    return {
        // ... existing state
        savedAnimations: writable([]),
        currentAnimationId: writable(null),
        showLoadModal: writable(false),
        showSaveModal: writable(false),
    };
}
```

### Phase 6: Frontend Components

#### 6.1 Create LoadAnimationModal.svelte

New component in `/front/src/lib/components/LoadAnimationModal.svelte`:

```svelte
<script lang="ts">
import { createEventDispatcher } from 'svelte';
import type { SavedAnimation } from '$lib/api/generated';

type Props = {
    animations: SavedAnimation[];
    open: boolean;
};

let { animations, open = $bindable() }: Props = $props();

const dispatch = createEventDispatcher<{
    load: string;  // animation ID
    delete: string;
}>();

function handleLoad(id: string) {
    dispatch('load', id);
    open = false;
}

function handleDelete(id: string) {
    dispatch('delete', id);
}
</script>

{#if open}
<div class="modal-backdrop" onclick={() => open = false}>
    <div class="modal-content" onclick={(e) => e.stopPropagation()}>
        <h2>Load Animation</h2>

        {#if animations.length === 0}
            <p>No saved animations for this device.</p>
        {:else}
            <ul class="animations-list">
                {#each animations as anim}
                    <li>
                        <div class="animation-item">
                            <span class="name">{anim.name}</span>
                            <span class="meta">{anim.frames.length} frames</span>
                            <div class="actions">
                                <button onclick={() => handleLoad(anim.id)}>
                                    Load
                                </button>
                                <button
                                    class="delete"
                                    onclick={() => handleDelete(anim.id)}
                                >
                                    Delete
                                </button>
                            </div>
                        </div>
                    </li>
                {/each}
            </ul>
        {/if}

        <button onclick={() => open = false}>Close</button>
    </div>
</div>
{/if}

<style>
/* Modal styling */
</style>
```

#### 6.2 Create SaveAnimationModal.svelte

New component in `/front/src/lib/components/SaveAnimationModal.svelte`:

```svelte
<script lang="ts">
import { createEventDispatcher } from 'svelte';

type Props = {
    open: boolean;
    currentAnimationName: string | null;  // If editing loaded animation
};

let { open = $bindable(), currentAnimationName }: Props = $props();

const dispatch = createEventDispatcher<{
    save: { name: string; overwrite: boolean };
}>();

let name = $state('');
let showOverwriteOption = $derived(currentAnimationName !== null);

function handleSave(overwrite: boolean) {
    if (!name.trim()) return;

    dispatch('save', { name: name.trim(), overwrite });
    name = '';
    open = false;
}
</script>

{#if open}
<div class="modal-backdrop" onclick={() => open = false}>
    <div class="modal-content" onclick={(e) => e.stopPropagation()}>
        <h2>Save Animation</h2>

        <label>
            Animation Name:
            <input type="text" bind:value={name} placeholder="Enter name..." />
        </label>

        <div class="actions">
            {#if showOverwriteOption}
                <button
                    onclick={() => handleSave(true)}
                    disabled={!name.trim()}
                >
                    Update "{currentAnimationName}"
                </button>
            {/if}
            <button
                onclick={() => handleSave(false)}
                disabled={!name.trim()}
            >
                Save as New
            </button>
            <button onclick={() => { open = false; name = ''; }}>
                Cancel
            </button>
        </div>
    </div>
</div>
{/if}
```

#### 6.3 Update +page.svelte

Add modal components and new buttons:

```svelte
<script lang="ts">
import LoadAnimationModal from '$lib/components/LoadAnimationModal.svelte';
import SaveAnimationModal from '$lib/components/SaveAnimationModal.svelte';
import { saveAnimation, updateAnimation, loadAnimation,
         deleteAnimation, listAnimations } from '$lib/api/mock';

// ... existing code

// New state
let showLoadModal = $state(false);
let showSaveModal = $state(false);
let savedAnimations = $state<SavedAnimation[]>([]);
let currentAnimationId = $state<string | null>(null);
let currentAnimationName = $state<string | null>(null);

// Load saved animations when device changes
$effect(() => {
    const device = get(selectedDevice);
    if (device) {
        refreshSavedAnimations(device.location);
    }
});

async function refreshSavedAnimations(deviceId: string) {
    savedAnimations = await listAnimations(deviceId);
}

async function handleSaveAnimation(event: CustomEvent<{ name: string; overwrite: boolean }>) {
    const { name, overwrite } = event.detail;
    const deviceLocation = get(selectedDevice)?.location;
    const framesList = get(frames);

    if (!deviceLocation) return;

    try {
        if (overwrite && currentAnimationId) {
            const updated = await updateAnimation(
                currentAnimationId,
                name,
                framesList.map(f => f.pixels)
            );
            currentAnimationName = updated.name;
        } else {
            const saved = await saveAnimation(
                deviceLocation,
                name,
                framesList.map(f => f.pixels)
            );
            currentAnimationId = saved.id;
            currentAnimationName = saved.name;
        }

        await refreshSavedAnimations(deviceLocation);
    } catch (e) {
        console.error('Failed to save:', e);
    }
}

async function handleLoadAnimation(event: CustomEvent<string>) {
    const animationId = event.detail;

    try {
        const animation = await loadAnimation(animationId);

        // Convert API frames to frontend Frame format
        const loadedFrames = animation.frames.map((apiFrame, i) => ({
            id: `frame-${Date.now()}-${i}`,
            name: `Frame ${i + 1}`,
            pixels: apiFrame.map(pixel => packRGB(pixel.r, pixel.g, pixel.b))
        }));

        frames.set(loadedFrames);
        currentAnimationId = animation.id;
        currentAnimationName = animation.name;
    } catch (e) {
        console.error('Failed to load:', e);
    }
}

async function handleDeleteAnimation(event: CustomEvent<string>) {
    const animationId = event.detail;

    try {
        await deleteAnimation(animationId);

        // Clear current animation if it was deleted
        if (currentAnimationId === animationId) {
            currentAnimationId = null;
            currentAnimationName = null;
        }

        const deviceLocation = get(selectedDevice)?.location;
        if (deviceLocation) {
            await refreshSavedAnimations(deviceLocation);
        }
    } catch (e) {
        console.error('Failed to delete:', e);
    }
}
</script>

<!-- ... existing UI ... -->

<!-- Apply Section - Add new buttons -->
<div class="apply-section">
    <!-- Existing Apply Animation button -->
    <button onclick={applyCurrentAnimation}>Apply Animation</button>
    <button onclick={stopCurrentAnimation}>Stop Animation</button>

    <!-- NEW BUTTONS -->
    <div class="storage-buttons">
        <button onclick={() => showSaveModal = true}>
            Save Animation
        </button>
        <button onclick={() => showLoadModal = true}>
            Load Animation
        </button>
    </div>

    <!-- Status messages... -->
</div>

<!-- Modals -->
<LoadAnimationModal
    bind:open={showLoadModal}
    animations={savedAnimations}
    onload={handleLoadAnimation}
    ondelete={handleDeleteAnimation}
/>

<SaveAnimationModal
    bind:open={showSaveModal}
    currentAnimationName={currentAnimationName}
    onsave={handleSaveAnimation}
/>
```

---

## Testing Checklist

1. **Backend Storage**
   - ✅ Database creates correctly on first run
   - ✅ Save animation creates new record
   - ✅ List animations returns only device's animations
   - ✅ Get animation retrieves correct data
   - ✅ Update animation modifies existing record
   - ✅ Delete animation removes record

2. **API Endpoints**
   - ✅ POST /animation/save returns 200 with animation object
   - ✅ GET /animation/list/{device_id} returns filtered list
   - ✅ GET /animation/{id} returns 404 for non-existent ID
   - ✅ PUT /animation/{id} updates correctly
   - ✅ DELETE /animation/{id} removes animation

3. **Frontend Flow**
   - ✅ Draw frames → Click "Save" → Enter name → Animation saved
   - ✅ Click "Load" → See list → Click animation → Frames loaded into editor
   - ✅ Load animation → Modify → Click "Save" → See "Update" and "Save as new" options
   - ✅ Select "Update" → Animation overwritten with same ID
   - ✅ Select "Save as new" → New animation created
   - ✅ Delete animation from modal → Removed from list
   - ✅ Switch devices → See different saved animations list
