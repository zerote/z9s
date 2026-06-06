# KT9S Setup Guide - Next Steps for You

**Date**: June 6, 2026  
**Status**: Initial structure created - ready for your continuation

---

## 📋 What's Been Done So Far

I've created the initial project structure for **z9s** on my Linux environment. Here's what you need to do next on your laptop:

### ✅ Completed
1. ✅ Analyzed both k9s and ktop codebases
2. ✅ Created detailed architecture document (`KT9S_ANALYSIS.md`)
3. ✅ Created project skeleton structure
4. ✅ Set up `go.mod` file (needs refinement)
5. ✅ Created `main.go` entry point
6. ✅ Created `cmd/root.go` with CLI flags
7. ✅ Created `internal/app/modes.go` for mode management
8. ✅ Created `internal/app/input_handler.go` for toggle logic
9. ✅ Created `README.md` project overview
10. ✅ Created `LICENSE` file (Apache 2.0)

### 📁 Current Project Structure

```
z9s/
├── cmd/
│   └── root.go              ← Main entry point with flags
├── internal/
│   └── app/
│       ├── modes.go         ← Mode definitions (k9s, ktop)
│       └── input_handler.go ← Ctrl+F10 toggle logic
├── main.go                  ← Application entry point
├── go.mod                   ← Go module file (needs go.sum)
├── LICENSE                  ← Apache 2.0
├── README.md                ← Project overview
├── KT9S_ANALYSIS.md         ← Detailed architecture (from my analysis)
└── go.mod.k9s / go.mod.ktop ← Original module files (reference)
```

---

## 🚀 Your Next Steps on Laptop

### Step 1: Create GitHub Repository

```bash
# Go to GitHub and create a new public repository named "z9s"
# Clone it locally
git clone https://github.com/YOUR_USERNAME/z9s.git
cd z9s
```

### Step 2: Get the Files I Created

**Option A: Download as ZIP** (Easier for mobile to laptop transfer)
1. I'll create a `z9s.zip` file with everything
2. Download it from here
3. Extract and copy to your laptop

**Option B: Copy Files Manually**
- Copy the files I created to your local z9s repo
- The key files are:
  - `main.go`
  - `go.mod`
  - `cmd/root.go`
  - `internal/app/modes.go`
  - `internal/app/input_handler.go`
  - `README.md`
  - `LICENSE`

### Step 3: Update go.mod

```bash
# Go to the z9s directory
cd z9s

# Initialize dependencies (this will download all required packages)
go mod tidy

# This may take a while the first time
# The actual go.mod I created was a simplified version
```

### Step 4: Try to Build

```bash
# Try building to see what's missing
go build -o z9s

# This will likely fail with "import not found" errors
# Don't worry - that's normal because we haven't copied k9s and ktop code yet
```

### Step 5: Merge k9s and ktop Code

This is the heavy lifting. You have two options:

#### Option A: Systematic Copy (Recommended)
```bash
# 1. Copy k9s code to internal/k9s/
cp -r /path/to/k9s-original/internal/* internal/k9s/
cp -r /path/to/k9s-original/cmd/* internal/k9s/cmd/

# 2. Copy ktop code to internal/ktop/
cp -r /path/to/ktop-original/* internal/ktop/

# 3. Fix import paths in both:
# - Change "github.com/derailed/k9s" to "github.com/YOUR_USERNAME/z9s/internal/k9s"
# - Change "github.com/vladimirvivien/ktop" to "github.com/YOUR_USERNAME/z9s/internal/ktop"
```

#### Option B: Git Subtree (Advanced)
```bash
# Add both repos as remotes
git remote add k9s-origin https://github.com/derailed/k9s.git
git remote add ktop-origin https://github.com/vladimirvivien/ktop.git

# Pull them as subtrees
git subtree add --prefix=internal/k9s k9s-origin main --squash
git subtree add --prefix=internal/ktop ktop-origin main --squash
```

### Step 6: Fix Import Paths

After copying, you'll need to update imports. Create a script to do this:

```bash
#!/bin/bash

# Fix k9s imports
find internal/k9s -name "*.go" -type f | xargs sed -i 's|github.com/derailed/k9s|github.com/YOUR_USERNAME/z9s/internal/k9s|g'

# Fix ktop imports  
find internal/ktop -name "*.go" -type f | xargs sed -i 's|github.com/vladimirvivien/ktop|github.com/YOUR_USERNAME/z9s/internal/ktop|g'
```

### Step 7: Update go.mod

```bash
# Make sure all dependencies are available
go mod tidy

# Download them
go mod download
```

### Step 8: Try Building Again

```bash
go build -o z9s
```

You'll likely get errors. Document them and we can fix them together.

---

## 💡 Implementation Path (Phases)

### Phase 1: Get it Compiling ✓ (You are here)
- [ ] Copy k9s code
- [ ] Copy ktop code
- [ ] Fix imports
- [ ] Get `go build` working

### Phase 2: Get it Running
- [ ] Create shared K8s client
- [ ] Adapt k9s's `view.App` to work in the new structure
- [ ] Adapt ktop's `application.App` to work in the new structure
- [ ] Get both modes to initialize

### Phase 3: Implement Toggle
- [ ] Wire up input handler
- [ ] Implement mode pause/resume
- [ ] Test Ctrl+F10 toggling
- [ ] Preserve state between switches

### Phase 4: Polish & Testing
- [ ] Fix bugs
- [ ] Add comprehensive testing
- [ ] Performance optimization
- [ ] Create proper documentation

---

## 🎯 Key Architecture Points to Remember

### Input Flow with Toggle
```
Input Event (Ctrl+F10?)
    ↓
InputHandler.HandleKeyEvent()
    ↓
If Ctrl+F10: ModeContext.Toggle()
    else: Forward to current mode
    ↓
Current Mode processes event
```

### Mode Structure
```
ModeContext {
    CurrentMode: int (ModeK9s or ModeKtop)
    K8sClient: shared instance
    Logger: shared instance
    Config: shared or mode-specific
}
```

### Files You Need to Create Eventually

1. **AppManager** (`internal/app/app_manager.go`)
   - Orchestrates both modes
   - Handles pause/resume
   - Manages the main loop

2. **Shared K8s Client** (`internal/shared/k8s_client/client.go`)
   - Unified interface for both modes
   - Lifecycle management

3. **Mode Adapters**
   - `internal/k9s/app.go` - Wrapper for k9s view.App
   - `internal/ktop/app.go` - Wrapper for ktop application.App

---

## 🤔 Quick Decisions to Make

Before you start intensive work, decide:

1. **Module name**: Use `github.com/YOUR_USERNAME/z9s` or something else?
2. **Default mode**: Start with k9s or ktop?
3. **Merge strategy**: Copy-paste or Git subtree or something else?
4. **Config files**: One combined config or separate?
5. **Versioning**: v0.1.0 or v1.0.0 as starting point?

---

## 🔗 Reference Documentation

### Files I Created
- **KT9S_ANALYSIS.md** - Complete architecture breakdown
- **README.md** - User-facing documentation

### Original Repos
- **k9s**: https://github.com/derailed/k9s
  - Key: `internal/view/app.go` - the main App struct
  - Key: `cmd/root.go` - how it initializes

- **ktop**: https://github.com/vladimirvivien/ktop
  - Key: `application/app_ctrl.go` - the main control loop
  - Key: `cmd/ktop.go` - how it initializes

---

## 📞 When You Get Stuck

Common issues and solutions:

### "Module not found" errors
```
Solution: Run `go mod tidy` to resolve dependencies
```

### Import path conflicts
```
Solution: Use sed/grep to find and replace all import statements
```

### Can't find functions from original k9s/ktop
```
Solution: Check if the import path is updated in go.mod
```

### Build succeeds but runtime panics
```
Solution: Check if modes are properly initialized in AppManager
```

---

## 📊 Progress Checklist

Use this to track your progress:

```
[ ] Created GitHub repo
[ ] Downloaded files from Claude
[ ] Copied k9s code to internal/k9s/
[ ] Copied ktop code to internal/ktop/
[ ] Fixed import paths
[ ] go mod tidy successful
[ ] go build successful
[ ] Binary runs without panic
[ ] Both modes initialize
[ ] Ctrl+F10 toggle works
[ ] State preserves between toggles
[ ] All tests pass
[ ] README updated
[ ] Ready for public release
```

---

## 🚀 Once You're Up and Running

After you get it building and running, we can focus on:

1. **Improving the toggle** - Maybe add visual feedback
2. **Shared components** - Extract common UI code
3. **Configuration** - Unified config file
4. **Plugins** - Make it extensible
5. **Performance** - Optimize the handoff between modes

---

## 💾 Sharing Between Devices

Remember the workflow:

1. **Work on Laptop** - Make code changes
2. **Commit to GitHub** - Push your changes
3. **Work on Mobile** - Pull latest from GitHub (use mobile GitHub app or clone)
4. **Back to Laptop** - Pull changes from GitHub

For small changes, you can also use cloud storage (Google Drive, Dropbox) to sync files.

---

## 📞 Final Notes

- This is a substantial project, so take it step by step
- Don't try to do everything at once
- Focus first on getting it to **compile**, then to **run**, then to **toggle**
- Document issues as you find them - they'll be valuable for improvements

Good luck! 🎉

---

**Last Updated**: June 6, 2026  
**Next Check-in**: After you've copied the k9s and ktop code
