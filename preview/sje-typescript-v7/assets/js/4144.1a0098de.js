"use strict";(self.webpackChunkzigflow=self.webpackChunkzigflow||[]).push([["4144"],{7454(e,t,a){function i(e,t){e.accDescr&&t.setAccDescription?.(e.accDescr),e.accTitle&&t.setAccTitle?.(e.accTitle),e.title&&t.setDiagramTitle?.(e.title)}a.d(t,{S:()=>i}),(0,a(6827).K)(i,"populateCommonDb")},5603(e,t,a){a.d(t,{diagram:()=>A});var i=a(7454),l=a(1475),r=a(6990),n=a(9150),s=a(1293),o=a(6827),c=a(8731),d=a(6936),p=a(4900),h=n.UI.pie,g={sections:new Map,showData:!1,config:h},u=g.sections,f=g.showData,m=structuredClone(h),w=(0,o.K)(()=>structuredClone(m),"getConfig"),$=(0,o.K)(()=>{u=new Map,f=g.showData,(0,n.IU)()},"clear"),x=(0,o.K)(({label:e,value:t})=>{if(t<0)throw Error(`"${e}" has invalid value: ${t}. Negative values are not allowed in pie charts. All slice values must be >= 0.`);u.has(e)||(u.set(e,t),s.R.debug(`added new section: ${e}, with value: ${t}`))},"addSection"),S=(0,o.K)(()=>u,"getSections"),y=(0,o.K)(e=>{f=e},"setShowData"),b=(0,o.K)(()=>f,"getShowData"),C={getConfig:w,clear:$,setDiagramTitle:n.ke,getDiagramTitle:n.ab,setAccTitle:n.SV,getAccTitle:n.iN,setAccDescription:n.EI,getAccDescription:n.m7,addSection:x,getSections:S,setShowData:y,getShowData:b},D=(0,o.K)((e,t)=>{(0,i.S)(e,t),t.setShowData(e.showData),e.sections.map(t.addSection)},"populateDb"),T={parse:(0,o.K)(async e=>{let t=await (0,c.qg)("pie",e);s.R.debug(t),D(t,C)},"parse")},k=(0,o.K)(e=>`
  .pieCircle{
    stroke: ${e.pieStrokeColor};
    stroke-width : ${e.pieStrokeWidth};
    opacity : ${e.pieOpacity};
  }
  .pieCircle.highlighted{
    scale: 1.05;
    opacity: 1;
  }
  .pieCircle.highlightedOnHover:hover{
    transition-duration: 250ms;
    scale: 1.05;
    opacity: 1;
  }
  .pieOuterCircle{
    stroke: ${e.pieOuterStrokeColor};
    stroke-width: ${e.pieOuterStrokeWidth};
    fill: none;
  }
  .pieTitleText {
    text-anchor: middle;
    font-size: ${e.pieTitleTextSize};
    fill: ${e.pieTitleTextColor};
    font-family: ${e.fontFamily};
  }
  .slice {
    font-family: ${e.fontFamily};
    fill: ${e.pieSectionTextColor};
    font-size:${e.pieSectionTextSize};
    // fill: white;
  }
  .legend text {
    fill: ${e.pieLegendTextColor};
    font-family: ${e.fontFamily};
    font-size: ${e.pieLegendTextSize};
  }
`,"getStyles"),v=(0,o.K)(e=>{let t=[...e.values()].reduce((e,t)=>e+t,0),a=[...e.entries()].map(([e,t])=>({label:e,value:t})).filter(e=>e.value/t*100>=1);return(0,d.rL)().value(e=>e.value).sort(null)(a)},"createPieArcs"),A={parser:T,db:C,renderer:{draw:(0,o.K)((e,t,a,i)=>{s.R.debug("rendering pie chart\n"+e);let o=i.db,c=(0,n.D7)(),h=(0,r.$t)(o.getConfig(),c.pie),g=(0,l.D)(t),u=g.append("g");u.attr("transform","translate(225,225)");let{themeVariables:f}=c,[m]=(0,r.I5)(f.pieOuterStrokeWidth);m??=2;let w=h.legendPosition,$=h.textPosition,x=h.donutHole>0&&h.donutHole<=.9?h.donutHole:0,S=(0,d.JL)().innerRadius(185*x).outerRadius(185),y=(0,d.JL)().innerRadius(185*$).outerRadius(185*$),b=u.append("g");b.append("circle").attr("cx",0).attr("cy",0).attr("r",185+m/2).attr("class","pieOuterCircle");let C=o.getSections(),D=v(C),T=[f.pie1,f.pie2,f.pie3,f.pie4,f.pie5,f.pie6,f.pie7,f.pie8,f.pie9,f.pie10,f.pie11,f.pie12],k=0;C.forEach(e=>{k+=e});let A=D.filter(e=>"0"!==(e.data.value/k*100).toFixed(0)),K=(0,p.UM)(T).domain([...C.keys()]);b.selectAll("mySlices").data(A).enter().append("path").attr("d",S).attr("fill",e=>K(e.data.label)).attr("class",e=>{let t="pieCircle";return"hover"===h.highlightSlice?t+=" highlightedOnHover":h.highlightSlice===e.data.label&&(t+=" highlighted"),t}),b.selectAll("mySlices").data(A).enter().append("text").text(e=>(e.data.value/k*100).toFixed(0)+"%").attr("transform",e=>"translate("+y.centroid(e)+")").style("text-anchor","middle").attr("class","slice");let R=u.append("text").text(o.getDiagramTitle()).attr("x",0).attr("y",-200).attr("class","pieTitleText"),z=[...C.entries()].map(([e,t])=>({label:e,value:t})),O=u.selectAll(".legend").data(z).enter().append("g").attr("class","legend");O.append("rect").attr("width",18).attr("height",18).style("fill",e=>K(e.label)).style("stroke",e=>K(e.label)),O.append("text").attr("x",22).attr("y",14).text(e=>o.getShowData()?`${e.label} [${e.value}]`:e.label);let M=Math.max(...O.selectAll("text").nodes().map(e=>e?.getBoundingClientRect().width??0)),F=450,H=490,L=22*z.length;switch(w){case"center":O.attr("transform",(e,t)=>"translate("+(-M/2-22)+","+(22*t-22*z.length/2)+")");break;case"top":F+=L,O.attr("transform",(e,t)=>`translate(${-M/2-22}, ${22*t-185})`),b.attr("transform",()=>`translate(0, ${L+22})`);break;case"bottom":F+=L,O.attr("transform",(e,t)=>"translate("+(-M/2-22)+","+(22*t- -207)+")");break;case"left":H+=22+M,O.attr("transform",(e,t)=>"translate(-207,"+(22*t-22*z.length/2)+")"),b.attr("transform",()=>`translate(${M+18+4}, 0)`);break;default:H+=22+M,O.attr("transform",(e,t)=>"translate(216,"+(22*t-22*z.length/2)+")")}let I=R.node()?.getBoundingClientRect().width??0,W=Math.min(0,225-I/2),B=Math.max(H,225+I/2)-W;g.attr("viewBox",`${W} 0 ${B} ${F}`),(0,n.a$)(g,F,B,h.useMaxWidth)},"draw")},styles:k}}}]);