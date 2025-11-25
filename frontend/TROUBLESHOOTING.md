# Troubleshooting Frontend Blank Screen

## Issue
Frontend shows blank screen at http://localhost:5173

## Steps to Fix

### 1. Hard Refresh Browser
Press **Cmd+Shift+R** (Mac) or **Ctrl+Shift+R** (Windows) to clear cache and reload.

### 2. Check Browser Console
1. Open Developer Tools (F12 or Cmd+Option+I)
2. Go to Console tab
3. Look for any red errors
4. Share the errors if you see any

### 3. Check Network Tab
1. Open Developer Tools → Network tab
2. Refresh page
3. Check if `main.tsx` and `App.tsx` are loading (should be 200 OK)

### 4. Verify Server is Running
```bash
# Check if dev server is running
curl http://localhost:5173/

# Should return HTML with <div id="root">
```

### 5. Clear Everything and Restart
```bash
# Stop dev server (Ctrl+C)
cd frontend

# Clear node modules
rm -rf node_modules package-lock.json

# Reinstall
npm install

# Start fresh
npm run dev
```

### 6. Check if React is Loading
Open browser console and type:
```javascript
document.getElementById('root')
```
Should return the div element, not null.

## Common Causes
- ❌ Browser cache (fix: hard refresh)
- ❌ React not mounting (check console errors)
- ❌ Routing issue (check URL - should be exactly http://localhost:5173/)
- ❌ CORS issue (backend should allow localhost:5173)

## What Should Work
1. Navigate to http://localhost:5173/
2. See welcome screen with:
   - ⚡ logo at top
   - "AI Knowledge System" in orange
   - "Create First Document" button
   - Sidebar on left

If still not working, please share:
1. Browser console errors (if any)
2. Network tab status
3. What URL you'reseeing in address bar
