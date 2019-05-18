# Overview
ratelimiter is a library to limit request rate

# Installing
<pre>
    <code>
    go get -u github.com/atuowgo/ratelimiter
    </code>
</pre>

Next, include in your application:
<pre>
    <code>
    import "github.com/atuowgo/ratelimiter"
    </code>
</pre>

# Using
<pre>
    <code>
    r := ratelimiter.Rule{
    		Name:         "rule1",
    		IntervalInMs: 1000,
    		CellNum:      2,
    		LimitQps:     200,
    }
    ratelimiter.Load(r)
    
    //before call the resource,call the Entry()
    succ,ctx := Entry()
    if succ {
        resource()
    }
    //finally call the Exit()
    Exit(ctx)
    
    
    
    func resource() {
        time.Sleep(time.Millisecond * time.Duration(rand.Intn(50)))
        resCnt++
    }
    
    </code>
</pre> 