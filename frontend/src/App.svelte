<script>
  import { Router, Route } from "svelte-routing";
  import Home from "./Home.svelte";
  import Admin from "./routes/Admin.svelte";
</script>

<Router>
  <Route path="/" component={Home} />
  <Route path="/admin" component={Admin} />
</Router>
