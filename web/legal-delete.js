'use strict';
window.addEventListener('DOMContentLoaded',()=>{
  const form=document.getElementById('delete-form');
  if(!form)return;
  form.addEventListener('submit',async e=>{
    e.preventDefault();
    if(!confirm('Permanently delete this FormForge account and active data?'))return;
    const out=document.getElementById('delete-result');
    try{
      const response=await fetch('/api/account-deletion/request',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(Object.fromEntries(new FormData(form)))});
      const data=await response.json().catch(()=>({}));
      if(!response.ok)throw new Error(data.message||'Deletion failed.');
      out.textContent='Account deleted. Remember to cancel any Apple or Google subscription separately.';
      form.remove();
    }catch(error){out.textContent=error.message||'Deletion failed.'}
  });
});
