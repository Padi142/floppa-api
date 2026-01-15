<script>
  import { onMount } from 'svelte';

  const ADMIN_PASSWORD = import.meta.env.VITE_ADMIN_PASSWORD;
  const AUTH_KEY = 'floppa_admin_auth';
  const API_BASE = '/api';

  let isAuthenticated = false;
  let password = "";
  let error = "";
  let apiError = "";
  
  let collections = [];
  let selectedCollection = null;
  let items = [];
  let loading = false;
  let uploading = false;
  let fileInput;
  let page = 1;
  let perPage = 50;
  let hasMore = true;
  let scrollContainer;
  let observer;

  onMount(() => {
    const stored = localStorage.getItem(AUTH_KEY);
    if (stored === 'true') {
      isAuthenticated = true;
      loadCollections();
    }
  });

  $: if (scrollContainer && hasMore && !loading && selectedCollection) {
    setupObserver();
  }

  function setupObserver() {
    if (observer) observer.disconnect();
    
    const sentinel = document.getElementById('scroll-sentinel');
    if (!sentinel) return;
    
    observer = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting && hasMore && !loading) {
        loadMore();
      }
    }, { rootMargin: '200px' });
    
    observer.observe(sentinel);
  }

  function login() {
    if (password === ADMIN_PASSWORD) {
      isAuthenticated = true;
      localStorage.setItem(AUTH_KEY, 'true');
      error = "";
      apiError = "";
      loadCollections();
    } else {
      error = "Incorrect password";
    }
  }

  function logout() {
    isAuthenticated = false;
    localStorage.removeItem(AUTH_KEY);
    collections = [];
    items = [];
    selectedCollection = null;
  }

  async function loadCollections() {
    loading = true;
    apiError = "";
    try {
      const response = await fetch(API_BASE + '/collections');
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const allCollections = await response.json();
      collections = allCollections.items?.filter(c => c.type === 'base' && c.name !== 'users' && !c.name.startsWith('_')) || 
                    allCollections.filter(c => c.type === 'base' && c.name !== 'users' && !c.name.startsWith('_')) || [];
      if (collections.length > 0) {
        selectCollection(collections[0]);
      }
    } catch (err) {
      console.error('Failed to load collections:', err);
      apiError = 'Failed to load collections: ' + (err.message || 'Unknown error');
    }
    loading = false;
  }

  async function selectCollection(collection) {
    selectedCollection = collection;
    page = 1;
    items = [];
    hasMore = true;
    loading = true;
    apiError = "";
    try {
      const response = await fetch(`${API_BASE}/collections/${collection.name}/records?page=${page}&perPage=${perPage}&sort=-views`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const data = await response.json();
      items = data.items || data || [];
      const totalItems = data.totalItems || 0;
      hasMore = items.length < totalItems;
    } catch (err) {
      console.error('Failed to load items:', err);
      apiError = 'Failed to load items: ' + (err.message || 'Unknown error');
      items = [];
    }
    loading = false;
    setTimeout(setupObserver, 100);
  }

  async function loadMore() {
    if (!selectedCollection || loading || !hasMore) return;
    
    const scrollTop = scrollContainer?.scrollTop || 0;
    
    page++;
    loading = true;
    try {
      const response = await fetch(`${API_BASE}/collections/${selectedCollection.name}/records?page=${page}&perPage=${perPage}&sort=-views`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const data = await response.json();
      const newItems = data.items || data || [];
      items = [...items, ...newItems];
      const totalItems = data.totalItems || items.length;
      hasMore = items.length < totalItems;
      
      requestAnimationFrame(() => {
        if (scrollContainer) {
          scrollContainer.scrollTop = scrollTop;
        }
      });
    } catch (err) {
      console.error('Failed to load more items:', err);
      page--;
    }
    loading = false;
    setTimeout(setupObserver, 100);
  }

  function getImageUrl(item) {
    if (!item.image || !item.id || !selectedCollection) return null;
    
    let filename;
    if (typeof item.image === 'string') {
      filename = item.image;
    } else if (Array.isArray(item.image) && item.image.length > 0) {
      filename = typeof item.image[0] === 'string' ? item.image[0] : item.image[0].filename || item.image[0];
    } else if (item.image && typeof item.image === 'object') {
      filename = item.image.filename || item.image.name || item.image;
    } else {
      return null;
    }
    
    return `${API_BASE}/files/${selectedCollection.name}/${item.id}/${filename}`;
  }

  async function uploadImage(event) {
    const files = event.target.files;
    if (!files || files.length === 0 || !selectedCollection) return;

    uploading = true;
    try {
      for (const file of files) {
        const formData = new FormData();
        formData.append('image', file);
        formData.append('views', '0');
        
        const response = await fetch(`${API_BASE}/collections/${selectedCollection.name}/records`, {
          method: 'POST',
          body: formData,
        });
        
        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(errorData.message || `HTTP ${response.status}: ${response.statusText}`);
        }
      }
      await selectCollection(selectedCollection);
    } catch (err) {
      console.error('Upload failed:', err);
      alert('Upload failed: ' + err.message);
    }
    uploading = false;
    if (fileInput) fileInput.value = '';
  }

  async function deleteItem(item) {
    if (!confirm('Delete this image?')) return;
    
    try {
      const response = await fetch(`${API_BASE}/collections/${selectedCollection.name}/records/${item.id}`, {
        method: 'DELETE',
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `HTTP ${response.status}: ${response.statusText}`);
      }
      
      items = items.filter(i => i.id !== item.id);
    } catch (err) {
      console.error('Delete failed:', err);
      alert('Delete failed: ' + err.message);
    }
  }
</script>

<div class="min-h-screen bg-zinc-950 text-white">
  {#if !isAuthenticated}
    <div class="min-h-screen flex items-center justify-center p-4">
      <div class="w-full max-w-md">
        <div class="bg-zinc-900 border border-zinc-800 rounded-2xl p-8">
          <div class="text-center mb-8">
            <div class="w-16 h-16 bg-gradient-to-br from-violet-500 to-fuchsia-500 rounded-2xl mx-auto mb-4 flex items-center justify-center">
              <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
            </div>
            <h1 class="text-2xl font-semibold">Admin Access</h1>
            <p class="text-zinc-500 mt-2">Enter password to continue</p>
          </div>
          
          <div class="space-y-4">
            <input
              type="password"
              bind:value={password}
              on:keydown={(e) => e.key === 'Enter' && login()}
              placeholder="Password"
              class="w-full px-4 py-3 bg-zinc-800 border border-zinc-700 rounded-xl focus:outline-none focus:border-violet-500 transition-colors"
            />
            
            {#if error}
              <p class="text-red-400 text-sm text-center">{error}</p>
            {/if}
            
            <button
              on:click={login}
              class="w-full py-3 bg-gradient-to-r from-violet-600 to-fuchsia-600 hover:from-violet-500 hover:to-fuchsia-500 rounded-xl font-medium transition-all"
            >
              Unlock
            </button>
          </div>
        </div>
      </div>
    </div>
  {:else}
    <div class="flex h-screen">
      <!-- Sidebar -->
      <aside class="w-72 bg-zinc-900 border-r border-zinc-800 flex flex-col">
        <div class="p-6 border-b border-zinc-800">
          <h1 class="text-xl font-semibold">Collections</h1>
          <p class="text-zinc-500 text-sm mt-1">Animal picture database</p>
        </div>
        
        <nav class="flex-1 overflow-y-auto p-3">
          {#each collections as collection}
            <button
              on:click={() => selectCollection(collection)}
              class="w-full text-left px-4 py-3 rounded-xl mb-1 transition-all {selectedCollection?.name === collection.name 
                ? 'bg-gradient-to-r from-violet-600/20 to-fuchsia-600/20 border border-violet-500/30' 
                : 'hover:bg-zinc-800'}"
            >
              <span class="font-medium capitalize">{collection.name}</span>
            </button>
          {/each}
          
          {#if collections.length === 0 && !loading}
            <p class="text-zinc-500 text-center py-8">No collections found</p>
          {/if}
        </nav>
        
        <div class="p-4 border-t border-zinc-800">
          <button
            on:click={logout}
            class="w-full py-2 text-zinc-400 hover:text-white hover:bg-zinc-800 rounded-xl transition-all text-sm"
          >
            Logout
          </button>
        </div>
      </aside>

      <!-- Main Content -->
      <main class="flex-1 flex flex-col overflow-hidden">
        {#if apiError}
          <div class="m-8 p-4 bg-red-500/10 border border-red-500/30 rounded-xl">
            <p class="text-red-400">{apiError}</p>
          </div>
        {/if}
        {#if selectedCollection}
          <!-- Header -->
          <header class="px-8 py-6 border-b border-zinc-800 flex items-center justify-between">
            <div>
              <h2 class="text-2xl font-semibold capitalize">{selectedCollection.name}</h2>
              <p class="text-zinc-500 mt-1">{items.length} images</p>
            </div>
            
            <div>
              <input
                type="file"
                accept="image/*"
                multiple
                bind:this={fileInput}
                on:change={uploadImage}
                class="hidden"
                id="file-upload"
              />
              <label
                for="file-upload"
                class="inline-flex items-center gap-2 px-5 py-2.5 bg-gradient-to-r from-violet-600 to-fuchsia-600 hover:from-violet-500 hover:to-fuchsia-500 rounded-xl font-medium cursor-pointer transition-all {uploading ? 'opacity-50 pointer-events-none' : ''}"
              >
                {#if uploading}
                  <svg class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  Uploading...
                {:else}
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                  </svg>
                  Upload
                {/if}
              </label>
            </div>
          </header>

          <!-- Image Grid -->
          <div class="flex-1 overflow-y-auto p-8" bind:this={scrollContainer}>
            {#if items.length === 0 && !loading}
              <div class="flex flex-col items-center justify-center h-64 text-zinc-500">
                <svg class="w-16 h-16 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <p class="text-lg">No images yet</p>
                <p class="text-sm mt-1">Upload some pictures to get started</p>
              </div>
            {:else}
              <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
                {#each items as item, index}
                  <div class="group relative bg-zinc-900 rounded-xl overflow-hidden border border-zinc-800 hover:border-zinc-700 transition-all">
                    <div class="aspect-square">
                      {#if getImageUrl(item)}
                        <img
                          src={getImageUrl(item)}
                          alt={item.id}
                          class="w-full h-full object-cover"
                          loading="lazy"
                        />
                      {:else}
                        <div class="w-full h-full bg-zinc-800 flex items-center justify-center">
                          <svg class="w-8 h-8 text-zinc-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                          </svg>
                        </div>
                      {/if}
                    </div>
                    
                    <!-- Overlay -->
                    <div class="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity">
                      <div class="absolute bottom-0 left-0 right-0 p-3">
                        <div class="flex items-center justify-between">
                          <div class="flex items-center gap-1.5 text-sm">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                            </svg>
                            {(item.views || 0).toLocaleString()}
                          </div>
                          <button
                            on:click={() => deleteItem(item)}
                            class="p-1.5 bg-red-500/20 hover:bg-red-500/40 rounded-lg transition-colors"
                          >
                            <svg class="w-4 h-4 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                          </button>
                        </div>
                      </div>
                    </div>

                    <!-- Rank Badge -->
                    {#if index < 3}
                      <div class="absolute top-2 left-2 w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold
                        {index === 0 ? 'bg-yellow-500 text-yellow-950' : index === 1 ? 'bg-zinc-300 text-zinc-800' : 'bg-amber-700 text-amber-100'}">
                        {index + 1}
                      </div>
                    {/if}
                  </div>
                {/each}
                <div id="scroll-sentinel"></div>
                {#if !hasMore && items.length > 0}
                  <div class="col-span-full flex items-center justify-center py-8 text-zinc-500 text-sm">
                    No more images
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        {:else}
          <div class="flex-1 flex items-center justify-center text-zinc-500">
            <p>Select a collection to view images</p>
          </div>
        {/if}
      </main>
    </div>
  {/if}
</div>
