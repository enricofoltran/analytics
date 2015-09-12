(function(){
    var d=document, i=new Image, e=encodeURIComponent;
    i.src='//{{ .Host }}/b.gif?u='+e(d.location.href)+'&r='+e(d.referrer)+'&t='+e(d.title);
})()
